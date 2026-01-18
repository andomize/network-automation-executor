package controller

import (
	"errors"
	"fmt"

	"github.com/andomize/network-automation-executor/internal/adapters/logger"
	"github.com/andomize/network-automation-executor/internal/core/domains"
)

/*
 * Controller.Autotests
 *
 * Метод Обрабатывает условия в рамках тестирования задачи
 */
func (c *Controller) Autotests() error {

	var testsResult error = nil

	if c.Task.Autotests != nil && len(*c.Task.Autotests) > 0 {
		logger.INFO("CTRL_AUTOTESTS: Start conditions autotesting")

		for testIdx, test := range *c.Task.Autotests {

			testPassed, testError := c.WhenMatcher(&[]domains.When{test}, c.Variables)

			if testError != nil {
				logger.INFO(fmt.Sprintf("CTRL_AUTOTESTS: TEST[%v] FAIL: %v", testIdx, testError))
				testsResult = errors.New(fmt.Sprintf("TEST[%v] FAIL: %v", testIdx, testError))
				continue
			}

			if !testPassed {
				logger.INFO(fmt.Sprintf("CTRL_AUTOTESTS: TEST[%v] FAIL: Condition failed", testIdx))
				testsResult = errors.New(fmt.Sprintf("TEST[%v] FAIL: %v", testIdx, testError))
				continue
			}

			if testPassed {
				logger.INFO(fmt.Sprintf("CTRL_AUTOTESTS: TEST[%v] PASSED", testIdx))
				continue
			}

			logger.INFO(fmt.Sprintf("CTRL_AUTOTESTS: TEST[%v] Unknown tests error", testIdx))
		}
	}

	return testsResult
}
