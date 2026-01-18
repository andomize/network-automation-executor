package controller

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
	"github.com/andomize/network-automation-executor/internal/core/ports"
)

/*
 * Controller.WhenMatcher
 *
 * Метод проверяет наличие условных выражений в задании и проверяет их корректность
 */
func (c *Controller) WhenMatcher(when *[]domains.When, vars Artefacts) (bool, error) {

	// Если условие есть, то необъодимо проверить что все требования соответствуют
	// и только после этого выполнять задание
	if when != nil && len(*when) > 0 {

		logger.DEBUG("WHEN_MATCH: Founded <when> field in task")
		logger.DEBUG("WHEN_MATCH: Starting match condition")

		// Условие существует - выполяем перебор всех условий и проверяем их
		for whenIdx, when := range *when {

			logger.DEBUG(fmt.Sprintf("WHEN_MATCH: Processing condition Idx: '%v'", whenIdx))

			// Проверяем что бы в условии не было одновременно несколько типов проверок
			if len(when.Name) > 0 && len(when.Variable) > 0 {
				logger.ERROR("WHEN_MATCH: WHEN::Name && WHEN::Variable is not allowed")
				return false, errors.New(ports.ERROR_WHEN_CONDITION_DOUBLE_BASED)
			}

			// Проверяем что требуется выполнить проверку по имени ранее выполненной задачи
			if len(when.Name) > 0 {

				logger.DEBUG("WHEN_MATCH: Condition have 'name' field, verifying")

				// Проверяем что имя задания существует в памяти
				if c.Names[when.Name] == nil || len(c.Names[when.Name].Status) <= 0 {
					logger.WARNING("WHEN_MATCH: WHEN::NAME Name '" + when.Name + "' does not exist in memory, skipping...")
					return false, nil
				}

				// IfStatus
				if len(when.IfStatus) > 0 {
					if c.Names[when.Name].Status != when.IfStatus {
						logger.WARNING("WHEN_MATCH: WHEN::NAME::IfStatus condition fail, want: '" + when.IfStatus + "', have: '" + c.Names[when.Name].Status + "'")
						return false, nil
					}
				}

				// IfOutputContains
				if len(when.IfOutputContains) > 0 {
					if !strings.Contains(c.Names[when.Name].Output, when.IfOutputContains) {
						logger.WARNING("WHEN_MATCH: WHEN::NAME::IfOutputContains condition fail, searched string: '" + when.IfOutputContains + "' in task name '" + when.Name + "'")
						return false, nil
					}
				}

				// IfOutputNotContains
				if len(when.IfOutputNotContains) > 0 {
					if strings.Contains(c.Names[when.Name].Output, when.IfOutputNotContains) {
						logger.WARNING("WHEN_MATCH: WHEN::NAME::IfOutputNotContains condition fail, searched string: '" + when.IfOutputNotContains + "' in task name '" + when.Name + "'")
						return false, nil
					}
				}

				// IfOutputNotContains
				if len(when.IfOutputContainsRe) > 0 {
					match, matchError := regexp.MatchString(when.IfOutputContainsRe, c.Names[when.Name].Output)
					if !match || matchError != nil {
						logger.WARNING("WHEN_MATCH: WHEN::NAME::IfOutputContainsRe condition fail, searched regexp: '" + when.IfOutputContainsRe + "' in task name '" + when.Name + "'")
						return false, nil
					}
				}

				// IfOutputNotContains
				if len(when.IfOutputNotContainsRe) > 0 {
					match, matchError := regexp.MatchString(when.IfOutputNotContainsRe, c.Names[when.Name].Output)
					if match && matchError != nil {
						logger.WARNING("WHEN_MATCH: WHEN::NAME::IfOutputNotContainsRe condition fail, searched regexp: '" + when.IfOutputNotContainsRe + "' in task name '" + when.Name + "'")
						return false, nil
					}
				}

				// Actions
				if actionError := c.WhenActions(&when); actionError != nil {
					return false, actionError
				}

				// Все условия успешно пройдены - выход
				logger.DEBUG("WHEN_MATCH: WHEN::NAME all condition stage is successful")
				return true, nil
			}

			// Проверяем что требуется выполнить проверку по одной из переменных
			if len(when.Variable) > 0 {

				logger.DEBUG("WHEN_MATCH: Condition have 'variables' field, verifying")

				// Проверяем что имя переменной существует в артефактах
				if len(vars[when.Variable]) <= 0 {
					logger.WARNING("WHEN_MATCH: WHEN::VARIABLE Variable '" + when.Variable + "' does not exist in memory, skipping...")
					return false, nil
				}

				// IfValue
				if len(when.IfValue) > 0 {
					if vars[when.Variable] != when.IfValue {
						logger.WARNING("WHEN_MATCH: WHEN::VARIABLE::IfValue condition fail, want: '" + when.IfValue + "', have: '" + vars[when.Variable] + "'")
						return false, nil
					}
				}

				// IfValueNot
				if len(when.IfValueNot) > 0 {
					if vars[when.Variable] == when.IfValueNot {
						logger.WARNING("WHEN_MATCH: WHEN::VARIABLE::IfValueNot condition fail: '" + when.IfValueNot + "' same as: '" + vars[when.Variable] + "'")
						return false, nil
					}
				}

				// Actions
				if actionError := c.WhenActions(&when); actionError != nil {
					return false, actionError
				}

				// Все условия успешно пройдены - выход
				logger.DEBUG("WHEN_MATCH: WHEN::VARIABLE all condition stage is successful")
				return true, nil
			}
		}
	}

	// Условных выражений в задании нету - всё ОК
	logger.DEBUG("WHEN_MATCH: Skipped match condition, no <when> field in task")
	return true, nil
}

/*
 * Controller.WhenActions
 *
 * Выполняет действия из условия
 */
func (c *Controller) WhenActions(when *domains.When) error {

	// nil verifying
	if when == nil {
		return errors.New("when is nil")
	}

	// OnExit
	if when.OnExit {
		logger.INFO("WHEN_ACTION: WHEN::OnExit is set, exiting...")
		c.ExitSuccess()
	}

	// OnMove
	if len(when.OnMove) > 0 {
		logger.INFO("WHEN_ACTION: WHEN::OnMove is set, next task name is '" + when.OnMove + "'")
		c.NextTaskName = when.OnMove
	}

	return nil
}
