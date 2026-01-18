package filestorage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

/*
 * FileStorage
 *
 * Описывает методы для работы с файлами в текущей файловой системе
 * Используется различными сервисами для того, что бы сохранять информацию
 * в файловой системе.
 * Обязательное условие инициализации - путь в пределах которого будет
 * происходить работа с файлами, за пределами указанной директории
 * работа с файлами невозможна, необходимо создавать новый экземпляр
 */
type FileStorage struct {
	// Путь к директории с файлами задания
	directory string

	// Разрешённые символы для имени файла
	AllowedRuneFilename string
}

func NewFileStorage(dir string) *FileStorage {
	return &FileStorage{
		directory:           dir,
		AllowedRuneFilename: `[a-zA-Z0-9-_\.]`,
	}
}

/*
 * FileStorage.Save
 *
 * Сохранение в файл
 */
func (f *FileStorage) Save(data []byte, filename string) error {

	if len(data) <= 0 {
		return nil
	}

	// Проверяем корректность имени файла
	if nameIsOk := f.NameVerify(filename); !nameIsOk {
		if normalize := f.NameNormalization(filename); len(normalize) > 0 {
			filename = normalize
		} else {
			return fmt.Errorf("Filename is incorrect: '%s'", filename)
		}
	}

	// Создаём путь к файлу, если он не был создан ранее
	filepath := filepath.Join(f.directory, filename)

	// Создаём путь к сохраняемому файлу, если он не существует
	errorCreateDir := os.MkdirAll(f.directory, os.ModePerm)
	if errorCreateDir != nil && !os.IsExist(errorCreateDir) {
		return errorCreateDir
	}

	// Create creates or truncates the named file.
	// If the file already exists, it is truncated.
	// If the file does not exist, it is created with mode 0666.
	// If successful, methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *PathError.
	file, errorCreateFile := os.Create(filepath)
	if errorCreateFile != nil {
		return errorCreateFile
	}
	defer file.Close()

	// Сохраняем в файл переданные для сохранения данные
	_, errorWrite := file.Write(data)
	if errorWrite != nil {
		return errorWrite
	}

	return nil
}

/* FileStorage.GetList
 *
 * Получение списка всех файлов в директории
 * Возвращает срез имён файлов
 */
func (f *FileStorage) GetList() ([]string, error) {

	// Инициализируем срез с именами файлов
	var fileslist []string

	// Открываем для чтения директорию с файлами
	filepath, err := os.Open(f.directory)
	if err != nil {
		return fileslist, err
	}
	defer filepath.Close()

	// Получаем информацию о файлах в открытой ранее директории
	filesinfo, err := filepath.Readdir(-1)
	if err != nil {
		return fileslist, err
	}

	// Перебираем список всех файлов и добавляем в возвращаемый
	// срез fileslist их имена
	for _, fileinfo := range filesinfo {
		fileslist = append(fileslist, fileinfo.Name())
	}
	return fileslist, nil
}

/* FileStorage.Read
 *
 * Прочитать содержимое отдельного файла
 */
func (f *FileStorage) Read(filename string) ([]byte, error) {

	// Объединение пути к файлам и имя файла
	filepath := filepath.Join(f.directory, filename)

	// Открытие файла
	file, err := os.Open(filepath)
	if err != nil {
		return []byte(""), err
	}
	defer file.Close()

	// Чтение содержимого файла
	filereader, err := ioutil.ReadAll(file)
	if err != nil {
		return []byte(""), err
	}

	// Возвращение содержимого файла
	return filereader, nil
}

/* FileStorage.GetDirectory
 *
 * Получить текущее расположение
 */
func (f *FileStorage) GetDirectory() string {
	return f.directory
}

/* FileStorage.NameVerify
 *
 * Проверить имя файла на инородные символы
 */
func (f *FileStorage) NameVerify(s string) bool {

	for _, symb := range s {
		matched, matchedError := regexp.MatchString(f.AllowedRuneFilename, string(symb))
		if !matched || matchedError != nil {
			return false
		}
	}

	return true
}

/* FileStorage.NameNormalization
 *
 * Удалить инородные символы из имени файла
 */
func (f *FileStorage) NameNormalization(s string) string {

	var result string = ""

	for _, symb := range s {
		matched, matchedError := regexp.MatchString(f.AllowedRuneFilename, string(symb))
		if matched && matchedError == nil {
			result += string(symb)
		}
	}

	return result
}
