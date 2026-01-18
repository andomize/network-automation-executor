package controller

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/ports"
)

/*
 * Controller.RegExpMatch
 *
 * Метод по заданному регулярному выражению парсит текст и возвращает карту со всеми
 * найденными группами и всеми вхождениями для каждой из групп
 */
func (c *Controller) RegExpMatch(
	output, incRegex, excRegex string) (map[string][]string, int, error) {

	logger.DEBUG("CTRL_REGIT: Include RegExp: '" + incRegex + "'")
	logger.DEBUG("CTRL_REGIT: Exclude RegExp: '" + excRegex + "'")

	// Компилируем полученное регулярное выражение
	compiledRegExp := regexp.MustCompile(incRegex)

	// SubexpNames возвращает имена заключенных в скобки подвыражений
	// в этом регулярном выражении. Имя для первого подвыражения — name[1],
	// поэтому, если m — срез совпадения, имя для m[i] — SubexpNames()[i].
	// Поскольку регулярное выражение в целом не может быть названо, name[0]
	// всегда является пустой строкой. Срез не должен изменяться.
	groupNames := compiledRegExp.SubexpNames()

	// Результат запишем в эту переменную
	var result = map[string][]string{}

	// Находим все совпадения для регулярного выражения в тексте и сохраняем их в result
	for _, match := range compiledRegExp.FindAllStringSubmatch(output, -1) {
		for groupIndex, group := range match {

			// При поиске подгруппы в тексте, к которому было применено регулярное выражение
			// первым (нулевым) элементом является вся найденная последовательность без
			// генерации подгрупп
			// Тем самым, нас интересуют только те значения, которые следуют после нулевого
			if groupIndex > 0 {

				// Вычленяем имя группы, если имя группы не задано, то оно будет пустым
				name := groupNames[groupIndex]

				// Проверяем было ли задано в регулярном выражении имя группы, которая была
				// обнаружена в текущей итерации (В формате "(?P<name>[a-zA-Z]+)")
				if len(name) <= 0 {
					// Если в явном виде не указано имя группы, то именем группы будет цифра
					// от 0 до n - позиция группы в регулярном выражении (номер скобки)
					name = strconv.Itoa(groupIndex)
				}

				// Добавляем найденный результат в список для возврата
				result[name] = append(result[name], group)
			}
		}
	}

	// Определим метод, находящий минимальное кол-во элементов
	get_min := func(slice map[string][]string) int {
		var min int = math.MaxInt
		for _, val := range slice {
			if len(val) < min {
				min = len(val)
			}
		}
		if min == math.MaxInt {
			return 0
		}
		return min
	}

	// Определим метод, находящий максимальное кол-во элементов
	get_max := func(slice map[string][]string) int {
		var max int = math.MinInt
		for _, val := range slice {
			if len(val) > max {
				max = len(val)
			}
		}
		if max == math.MinInt {
			return 0
		}
		return max
	}

	// Сравним кол-во элементов в группах и определим не потерялось ли чего
	// В группах должно быть одинаковое количество элементов, иначе корректное назначение
	// их на подзадание будет невозможно
	if get_min(result) != get_max(result) {
		// Если кол-во элементов в группах не совпадает, то ошибка
		logger.ERROR(ports.ERROR_REGEX_GROUP_NE)
		return nil, -1, errors.New("Count of element in different groups is not equal, max: " +
			strconv.Itoa(get_max(result)) + ", min: " + strconv.Itoa(get_min(result)) + ", st 1")
	}

	// Проверяем были ли переданы выражения, по которым нужно исключить результаты
	// из найденных групп в прошлом блоке кода
	if len(excRegex) > 0 {
		excludeRegExp := regexp.MustCompile(excRegex)

		// Записываем все индексы (номера элементов), которые нужно удалить из каждой группы
		var removingValueIndexPerGroups = []int{}

		// Переберём все найденные ранее группы
		for _, group := range result {

			// Переберём все найденные в группах значения
			for valueIndex, value := range group {

				// Проверим попадает ли найденное нами значение под синтаксис регулярного выражения
				// которое мы получили в переменной "excludeRegExp". Если совпадает, то все элементы
				// во всех группах с этим индексом будут удалены
				if excludeRegExp.MatchString(value) {
					logger.DEBUG("CTRL_REGIT: Removing index '" + strconv.Itoa(valueIndex) +
						"' is planed")
					removingValueIndexPerGroups = append(removingValueIndexPerGroups, valueIndex)
				}
			}
		}

		if len(removingValueIndexPerGroups) > 0 {
			logger.DEBUG("CTRL_REGIT: Group ready to removing: " + fmt.Sprint(result))

			// Исключаем те элементы, которые должны быть исключены
			for groupIdx, group := range result {

				// Помещаем сюда выжившие элементы после их удаления
				var survivors = []string{}

				// Переберём все найденные в группах значения
				for valueIndex, value := range group {

					// Разрешение на то, что бы этот элемент остался в группе
					var allowToAppend = true

					// Смотрим все индексы, которые нужно удалить
					for _, removingValueIndex := range removingValueIndexPerGroups {
						// Если индекс совпадает - не добавляем элемент в результат
						if valueIndex == removingValueIndex {
							logger.DEBUG("CTRL_REGIT: Removing item '" + value +
								"' from slice, index: '" + strconv.Itoa(valueIndex) + "'" +
								" using result by  death slice")
							allowToAppend = false
						}
					}

					if allowToAppend {
						logger.DEBUG("CTRL_REGIT: Item '" + value +
							"' from slice, index: '" + strconv.Itoa(valueIndex) + "is still alive")
						survivors = append(survivors, value)
					}
				}

				// Присваиваем результату переформированную группу значений
				result[groupIdx] = survivors
			}
		}
	}

	// Сравним кол-во элементов в группах и определим не потерялось ли чего
	if get_min(result) != get_max(result) {
		// Если кол-во элементов в группах не совпадает, то ошибка
		logger.ERROR(ports.ERROR_REGEX_GROUP_NE)
		return nil, -1, errors.New("Count of element in different groups is not equal, max: " +
			strconv.Itoa(get_max(result)) + ", min: " + strconv.Itoa(get_min(result)) + ", st 2")
	}

	logger.DEBUG("CTRL_REGIT: Result group: " + fmt.Sprint(result))
	return result, get_min(result), nil
}

/*
 * Controller.RegExpConstructor
 *
 * Метод преобразует переменные, указанные в JSON строке в формате {{name}} в
 * соответтсвующие этой переменной значению. Если такой переменной нету - ошибка
 *
 * Пример работы:
 *  строка:      show ip route vrf {{vrfname}}
 *  артефакты:   map[vrfname: big-data]
 *  результат:   show ip route vrf big-data
 */
func (c *Controller) RegExpConstructor(text string, variables Artefacts) (string, error) {

	// Компилируем искомое регулярное выражение, которое мы хотим найти в тексте
	compiledRegExp := regexp.MustCompile(`{{(?P<variable>[^\s\t]+?)}}`)

	// Пройдёмся по всем найденным подстрокам и заменим их соответствующим значением
	for _, match := range compiledRegExp.FindAllStringSubmatch(text, -1) {

		// Если в артефактах существует значение с найденным в тексте ключом
		if newValue := variables[match[1]]; newValue != "" {
			// Заменяем всё подвыражение {{name}} на значение переменной
			text = strings.Replace(text, match[0], newValue, -1)
		} else {
			logger.ERROR(ports.ERROR_REGEX_VAR_NOT_EXIST)
			return text, errors.New(fmt.Sprintf("Text contains variable '%s', but"+
				" suited variable do not exist", match[0]))
		}
	}

	return text, nil
}
