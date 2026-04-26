package providers

import (
	"github.com/imbecility/mtproxy_parser/types"
	"log"
	"time"
)

// Registry хранит список всех зарегистрированных провайдеров
var Registry []types.Provider

// Register вызывается в init() каждого провайдера для добавления его в общий пул
func Register(name string, fetchFunc func() []string) {
	Registry = append(Registry, types.Provider{
		Name:  name,
		Fetch: fetchFunc,
	})
}

// withRetry реализует механизм повторных попыток для функции получения данных,
// возвращает обертку, которая перезапускает обход провайдера при пустом результате с увеличивающейся задержкой.
func withRetry(name string, fetch func() []string, attempts int) func() []string {
	return func() []string {
		for i := range attempts {
			result := fetch()
			if len(result) > 0 {
				return result
			}
			log.Printf(" [%s] попытка %d/%d не дала результатов, повтор...", name, i+1, attempts)
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
		log.Printf(" [%s] все %d попытки исчерпаны", name, attempts)
		return nil
	}
}
