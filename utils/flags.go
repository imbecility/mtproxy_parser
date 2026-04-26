package utils

import (
	"flag"
	"os"
	"strings"
)

// OptionalString инкапсулирует строковое значение флага
// и метаданные о его присутствии в аргументах командной строки.
type OptionalString struct {
	Value    string
	IsSet    bool
	HasValue bool
}

// String возвращает строковое представление значения флага
func (o *OptionalString) String() string {
	return o.Value
}

// Set устанавливает значение флага и обновляет внутренние индикаторы присутствия данных
func (o *OptionalString) Set(s string) error {
	o.IsSet = true
	o.Value = s
	o.HasValue = true
	return nil
}

// getAllFlagNames формирует карту имен всех зарегистрированных флагов для их идентификации в общем списке аргументов
func getAllFlagNames() map[string]struct{} {
	names := make(map[string]struct{})
	flag.VisitAll(func(f *flag.Flag) {
		names[f.Name] = struct{}{}
	})
	return names
}

// ResolveOptionalFlags анализирует аргументы командной строки для определения фактического состояния флага,
// позволяет корректно обработать случаи, когда флаг указан без значения или предшествует другому флагу.
func ResolveOptionalFlags(o *OptionalString, flagName string) {
	args := os.Args[1:]
	flagNames := getAllFlagNames()

	for i, arg := range args {
		normalized := strings.TrimLeft(arg, "-")

		// пропуск если это не требуемый флаг или если значение уже в самом аргументе (-validate=value)
		if normalized != flagName || strings.Contains(arg, "=") {
			continue
		}

		o.IsSet = true

		if i+1 >= len(args) {
			o.HasValue = false
			return
		}

		next := args[i+1]

		// проверки есть ли флаг в каком-либо виде в реестре флагов
		nextNormalized := strings.TrimLeft(next, "-")
		_, isFlag := flagNames[nextNormalized]
		nextBase := strings.SplitN(nextNormalized, "=", 2)[0]
		_, isFlagWithValue := flagNames[nextBase]

		if isFlag || isFlagWithValue {
			o.HasValue = false
			return
		}

		o.HasValue = true
		return
	}
}
