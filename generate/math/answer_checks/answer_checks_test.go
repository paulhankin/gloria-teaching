package answerchecks

import (
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

		if !strings.Contains(page, `class="answer-blank"`) {
			t.Errorf("question %d does not contain an empty answer area", i+1)
		}
	}
}
