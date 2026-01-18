package jsontask

import (
	"encoding/json"
	"io/ioutil"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
)

/*
 * Read
 *
 * Прочитать файл задания
 */
func Read(filepath string) (*domains.TaskPattern, error) {

	logger.DEBUG("JSON_TASK_READ: Starting read file: '" + filepath + "'")

	// Создаём экземпляр будущего задания
	var taskPattern domains.TaskPattern

	// Открываем файл для чтения и извлекаем его содержимое
	taskContent, readFileError := ioutil.ReadFile(filepath)
	if readFileError != nil {
		logger.ERROR("JSON_TASK_READ: Cannot read file: '" + filepath + "'")
		return nil, readFileError
	}

	// Преобразуем прочитанные данные в объект
	unmarshallError := json.Unmarshal(taskContent, &taskPattern)
	if unmarshallError != nil {
		logger.ERROR("JSON_TASK_READ: Cannot unmarshall file: '" + filepath + "'")
		return nil, unmarshallError
	}

	return &taskPattern, nil
}

/*
 * Write
 *
 * Сохранить файл задания
 */
func Write(filepath string, task domains.TaskPattern) error {

	logger.DEBUG("JSON_TASK_WRITE: Start saving file: '" + filepath + "'")
	// Struct values encode as JSON objects.
	// Each exported struct field becomes a member
	// of the object, using the field name as the
	// object key, unless the field is omitted.
	//
	// MarshalIndent is like Marshal but applies
	// Indent to format the output. Each JSON element
	// in the output will begin on a new line beginning
	// with prefix followed by one or more copies of
	// indent according to the indentation nesting.
	taskJSON, err := json.MarshalIndent(task, "", "    ")
	if err != nil {
		logger.ERROR("JSON_TASK_WRITE: Cannot marshal data: '" + filepath + "'")
		return err
	}

	// WriteFile writes data to a file named by
	// filename. If the file does not exist,
	// WriteFile creates it with permissions perm
	// (before umask); otherwise WriteFile truncates
	// it before writing, without changing permissions.
	err = ioutil.WriteFile(filepath, taskJSON, 0644)
	if err != nil {
		logger.ERROR("JSON_TASK_WRITE: Cannot write data: '" + filepath + "'")
		return err
	}

	return nil
}
