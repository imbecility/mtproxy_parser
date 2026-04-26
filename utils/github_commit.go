package utils

import (
	"context"
	"errors"
	"fmt"
	"github.com/imbecility/mtproxy_parser/config"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v85/github"
)

// RepoInfo хранит владельца и название репо
type RepoInfo struct {
	Owner string
	Name  string
}

// UploadConfig определяет параметры для процесса загрузки файлов в удаленный репозиторий.
type UploadConfig struct {
	Token      string // обязательно
	RepoUrl    string // обязательно, но любой формат
	Branch     string // не обязательно
	FilePath   string // обязательно
	RemotePath string // не обязательно
	CommitMsg  string // не обязательно
	Squash     bool   // не обязательно
}

// GitHubUploader определяет настроенные загрузчик на Github
type GitHubUploader struct {
	client *github.Client
	owner  string
	repo   string
	branch string
	email  string
}

// Upload выполняет высокоуровневую процедуру выгрузки файла на основе переданного UploadConfig:
// оркестрирует процесс парсинга параметров, инициализации клиента и опционального сжатия истории
func Upload(cfg UploadConfig) error {
	if cfg.Token == "" {
		value := os.Getenv("GITHUB_TOKEN")
		if value == "" {
			return fmt.Errorf("токен не задан: ни в 'UploadConfig.Token', ни в переменной окружения 'GITHUB_TOKEN'")
		}
		cfg.Token = value
	}

	repoInfo, err := ParseRepoURL(cfg.RepoUrl)
	if err != nil {
		return err
	}

	if cfg.FilePath == "" {
		return fmt.Errorf("путь для выгрузки 'UploadConfig.FilePath' не может быть пустой строкой")
	}
	if !IsFile(cfg.FilePath) {
		return fmt.Errorf("в 'UploadConfig.FilePath' должна быть передана строка с путем к файлу, а не к папке: '%q'", cfg.FilePath)
	}

	if cfg.Branch == "" {
		cfg.Branch = "main"
	}

	if cfg.RemotePath == "" {
		cfg.RemotePath = filepath.Base(cfg.FilePath)
	}

	userTime := time.Now().In(config.TimeZone)
	commitMsg := fmt.Sprintf("updated: %s", userTime.Format("02.01.2006 15:04:05"))

	if cfg.CommitMsg == "" {
		cfg.CommitMsg = commitMsg
	}

	uploader, err := NewGitHubUploader(
		cfg.Token,
		repoInfo.Owner,
		repoInfo.Name,
		cfg.Branch, // "main"
	)
	if err != nil {
		return fmt.Errorf("не удалось инициализировать выгрузку в репо: %v", err)
	}

	ctx := context.Background()

	if err := uploader.UploadFile(
		ctx,
		cfg.FilePath,   // локальный файл
		cfg.RemotePath, // путь в репо
		commitMsg,
	); err != nil {
		return fmt.Errorf("не удалось выгрузить файл в репо: %v", err)
	}

	if cfg.Squash {
		if err := uploader.SuperSquash(ctx, cfg.CommitMsg); err != nil {
			return fmt.Errorf("SuperSquash error: %v", err)
		}
	}

	direct := fmt.Sprintf("прямая ссылка на файл:\nhttps://raw.githubusercontent.com/%s/%s/refs/heads/%s/%s",
		repoInfo.Owner, repoInfo.Name, cfg.Branch, cfg.RemotePath)
	log.Printf(direct)
	return nil
}

// NewGitHubUploader инициализирует новый экземпляр загрузчика с проверкой обязательных параметров.
func NewGitHubUploader(token, owner, repo, branch string) (*GitHubUploader, error) {
	if token == "" {
		return nil, errors.New("токен github пуст")
	}
	if owner == "" || repo == "" || branch == "" {
		return nil, errors.New("owner, repo и branch не могут быть пусты")
	}

	return &GitHubUploader{
		client: github.NewClient(nil).WithAuthToken(token),
		owner:  owner,
		repo:   repo,
		branch: branch,
	}, nil
}

// GetEmail извлекает адрес электронной почты пользователя для формирования метаданных коммита:
// пытается получить основной верифицированный email или генерирует технический адрес GitHub.
func (u *GitHubUploader) GetEmail(ctx context.Context) string {
	user, _, err := u.client.Users.Get(ctx, "")
	if err != nil {
		return ""
	}

	if user.GetEmail() != "" {
		return user.GetEmail()
	}

	emails, _, err := u.client.Users.ListEmails(ctx, nil)
	if err == nil {
		for _, e := range emails {
			if e.GetPrimary() && e.GetVerified() && e.GetEmail() != "" {
				return e.GetEmail()
			}
		}
		for _, e := range emails {
			if e.GetEmail() != "" {
				return e.GetEmail()
			}
		}
	}

	if user.GetID() != 0 && user.GetLogin() != "" {
		return fmt.Sprintf("%d+%s@users.noreply.github.com", user.GetID(), user.GetLogin())
	}

	if user.GetLogin() != "" {
		return fmt.Sprintf("%s@users.noreply.github.com", user.GetLogin())
	}

	return ""
}

// UploadFile загружает или обновляет содержимое файла в целевом репозитории,
// автоматически вычисляет SHA существующего файла для выполнения операции обновления.
func (u *GitHubUploader) UploadFile(ctx context.Context, localPath, remotePath, commitMsg string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать %q: %w", localPath, err)
	}

	if err := u.checkRateLimit(ctx); err != nil {
		return err
	}

	if u.email == "" {
		u.email = u.GetEmail(ctx)
	}
	if u.email == "" {
		return errors.New("не удалось получить email для коммита")
	}

	opts := &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMsg),
		Content: content,
		Branch:  github.Ptr(u.branch),
		Committer: &github.CommitAuthor{
			Name:  github.Ptr(u.owner),
			Email: github.Ptr(u.email),
		},
	}

	fileContent, _, _, err := u.client.Repositories.GetContents(
		ctx,
		u.owner,
		u.repo,
		remotePath,
		&github.RepositoryContentGetOptions{Ref: u.branch},
	)

	switch {
	case err == nil && fileContent != nil:
		opts.SHA = fileContent.SHA
		_, _, err = u.client.Repositories.UpdateFile(ctx, u.owner, u.repo, remotePath, opts)
		if err != nil {
			return fmt.Errorf("не удалось обновить файл в репо %q: %w", remotePath, err)
		}

	case isNotFound(err):
		_, _, err = u.client.Repositories.CreateFile(ctx, u.owner, u.repo, remotePath, opts)
		if err != nil {
			return fmt.Errorf("не удалось создать файл в репо %q: %w", remotePath, err)
		}

	default:
		return fmt.Errorf("не удалось получить информацию о %q в репо: %w", remotePath, err)
	}

	return nil
}

// SuperSquash необратимо сжимает всю историю ветки в один orphan-коммит:
// удаляет всю историю изменений, оставляя только текущее состояние дерева файлов.
func (u *GitHubUploader) SuperSquash(ctx context.Context, commitMsg string) error {
	if err := u.checkRateLimit(ctx); err != nil {
		return err
	}

	ref, _, err := u.client.Git.GetRef(
		ctx,
		u.owner,
		u.repo,
		"refs/heads/"+u.branch,
	)
	if err != nil {
		return fmt.Errorf("не удалось получить ref бранча %q: %w", u.branch, err)
	}

	headSHA := ref.GetObject().GetSHA()
	if headSHA == "" {
		return errors.New("HEAD SHA пуст")
	}

	headCommit, _, err := u.client.Git.GetCommit(ctx, u.owner, u.repo, headSHA)
	if err != nil {
		return fmt.Errorf("не удалось получить HEAD коммита %q: %w", headSHA, err)
	}

	treeSHA := headCommit.GetTree().GetSHA()
	if treeSHA == "" {
		return errors.New("SHA дерева пуст")
	}

	commits, _, err := u.client.Repositories.ListCommits(
		ctx,
		u.owner,
		u.repo,
		&github.CommitsListOptions{
			SHA:         u.branch,
			ListOptions: github.ListOptions{PerPage: 2},
		},
	)
	if err != nil {
		return fmt.Errorf("список коммитов: %w", err)
	}

	if len(commits) < 2 {
		return nil
	}

	// новый orphan-коммит (без родителей) поверх текущего дерева
	// CreateCommit принимает значение (не указатель) github.Commit
	newCommit, _, err := u.client.Git.CreateCommit(
		ctx,
		u.owner,
		u.repo,
		github.Commit{
			Message: github.Ptr(commitMsg),
			Tree: &github.Tree{
				SHA: github.Ptr(treeSHA),
			},
			Author: &github.CommitAuthor{
				Name:  github.Ptr(u.owner),
				Email: github.Ptr(u.email),
				Date:  &github.Timestamp{Time: time.Now()},
			},
		},
		&github.CreateCommitOptions{},
	)
	if err != nil {
		return fmt.Errorf("не удалось создать новый orphan-коммит: %w", err)
	}

	// force-push: сдвиг ref ветки на новый коммит
	_, _, err = u.client.Git.UpdateRef(
		ctx,
		u.owner,
		u.repo,
		"refs/heads/"+u.branch,
		github.UpdateRef{
			SHA:   newCommit.GetSHA(),
			Force: github.Ptr(true),
		},
	)
	if err != nil {
		return fmt.Errorf("не удалось выполнить force-push для бранча %q: %w", u.branch, err)
	}

	return nil
}

// checkRateLimit проверяет доступный лимит запросов к GitHub API
func (u *GitHubUploader) checkRateLimit(ctx context.Context) error {
	limits, _, err := u.client.RateLimit.Get(ctx)
	if err != nil {
		return nil
	}
	core := limits.GetCore()
	if core.Remaining == 0 {
		resetIn := time.Until(core.Reset.Time)
		return fmt.Errorf("превышен лимит запросов github, сброс через %s", resetIn.Round(time.Second))
	}
	return nil
}

// isNotFound проверяет, является ли ошибка ответом 404 от GitHub.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok {
		return ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound
	}
	return false
}

// parseSSH извлекает из SSH-адреса репозитория RepoInfo.Owner и RepoInfo.Name,
// если формат адреса не содержит разделителя пути - вернет ошибку.
func parseSSH(rawURL string) (RepoInfo, error) {
	// git@github.com:owner/repo.git -> owner/repo
	colonIdx := strings.LastIndex(rawURL, ":")
	if colonIdx == -1 {
		return RepoInfo{}, fmt.Errorf("некорректный SSH URL: %q", rawURL)
	}

	path := rawURL[colonIdx+1:]
	return extractFromPath(path)
}

// parseHTTPS разбирает HTTPS-ссылку на репозиторий для получения RepoInfo.Owner и RepoInfo.Name,
// если ссылка невалидна для url.Parse - вернет ошибку.
func parseHTTPS(rawURL string) (RepoInfo, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return RepoInfo{}, fmt.Errorf("не удалось распарсить URL %q: %w", rawURL, err)
	}

	return extractFromPath(u.Path)
}

// extractFromPath извлекает RepoInfo.Owner и RepoInfo.Name из строкового пути,
// очищает путь от расширения .git и лишних слешей перед сегментацией.
func extractFromPath(path string) (RepoInfo, error) {
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return RepoInfo{}, fmt.Errorf("не удалось извлечь владельца и репо из %q", path)
	}
	return buildRepoInfo(parts[0], parts[1])
}

// buildRepoInfo валидирует входные данные и формирует структуру RepoInfo,
// удаляет лишние пробелы и проверяет обязательные поля на наличие содержимого.
func buildRepoInfo(owner, name string) (RepoInfo, error) {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)

	if owner == "" {
		return RepoInfo{}, fmt.Errorf("владелец репо пуст")
	}
	if name == "" {
		return RepoInfo{}, fmt.Errorf("имя репо пусто")
	}

	return RepoInfo{Owner: owner, Name: name}, nil
}

// ParseRepoURL парсит ссылку на репозиторий и возвращает владельца и название репо из строк вида:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - git@github.com:owner/repo.git
//   - owner/repo
func ParseRepoURL(rawURL string) (RepoInfo, error) {
	rawURL = strings.TrimSpace(rawURL)
	// owner/repo
	if !strings.Contains(rawURL, "://") && !strings.Contains(rawURL, "@") {
		parts := strings.Split(strings.Trim(rawURL, "/"), "/")
		if len(parts) == 2 {
			return buildRepoInfo(parts[0], parts[1])
		}
	}
	// SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") || strings.Contains(rawURL, "@") && !strings.Contains(rawURL, "://") {
		return parseSSH(rawURL)
	}
	// HTTPS: https://github.com/owner/repo
	return parseHTTPS(rawURL)
}

// IsFile проверяет существование пути и что он файл, а не папка
func IsFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}
