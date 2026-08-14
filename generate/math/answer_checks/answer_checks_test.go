package answerchecks

import (
	"fmt"
	"strings"
	"testing"
)

func TestArithmeticQuestionsDoNotRevealAnswersOrShowAnswerCheckPrompt(t *testing.T) {
	for i, task := range arithmeticTasks {
		page := arithmeticPage(i+1, task)

		for _, unwanted := range []string{
			"Kann das stimmen?",
			"□ Ja",
			"□ Nein",
			`class="claim"`,
		} {
			if strings.Contains(page, unwanted) {
				t.Errorf("question %d contains %q", i+1, unwanted)
			}
		}

		if strings.Contains(page, fmt.Sprintf(">%d<", task.Correct)) {
			t.Errorf("question %d reveals the solution %d", i+1, task.Correct)
		}
		if !strings.Contains(page, `<div class="answer-blank"></div>`) {
			t.Errorf("question %d does not contain an empty answer area", i+1)
		}
	}
}
