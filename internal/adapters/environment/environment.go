package environment

import (
	"log"
	"os"
	"strings"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
)

func Get(name string, defaultValue string, panic bool) string {

	// Извлекаем переменную окружения
	value := os.Getenv(name)

	if value == "" {
		// Переменная не существует
		// Если установлено значение panic=true, то закрываем программу с ошибкой
		if panic {
			log.Fatal("Environment '" + name + "' is not set." +
				" Please use command 'export " + name + "=value' to fix this error on linux" +
				" or 'set " + name + "=value' on windows")

		}

		// Если есть значение по умолчанию, то используем его и возвращаем результат
		value = defaultValue
	}

	// Проверяем что извлекаемый нами токен не содержит таких слов, как "password", "token",
	// что бы предотвратить вывод в консоль значений и не раскрыть конфиденциальную информацию
	if strings.Contains(strings.ToLower(name), "password") ||
		strings.Contains(strings.ToLower(name), "token") {
		logger.DEBUG("Reading environment '" + name + "' successful. The value is hidden.")
	} else {
		logger.DEBUG("Reading environment '" + name + "' = '" + value + "'")
	}

	return value
}
