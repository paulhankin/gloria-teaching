package negativenumbers

type subtractionTask struct {
	Minuend    int
	Subtrahend int
}

func (t subtractionTask) result() int {
	return t.Minuend - t.Subtrahend
}

type taskGroup struct {
	Title string
	Hint  string
	Tasks []subtractionTask
}

// Task groups progress from one step below zero to larger negative results.
// Pupil-facing text stays German on purpose.
var taskGroups = []taskGroup{
	{
		Title: "1. Ein Schritt unter null",
		Hint:  "Gehe jeweils einen Schritt weiter als bis zur 0.",
		Tasks: []subtractionTask{
			{Minuend: 4, Subtrahend: 5},
			{Minuend: 2, Subtrahend: 3},
			{Minuend: 6, Subtrahend: 7},
			{Minuend: 1, Subtrahend: 2},
			{Minuend: 8, Subtrahend: 9},
			{Minuend: 3, Subtrahend: 4},
			{Minuend: 5, Subtrahend: 6},
			{Minuend: 7, Subtrahend: 8},
		},
	},
	{
		Title: "2. Zwei Schritte unter null",
		Hint:  "Gehe nun zwei Schritte weiter als bis zur 0.",
		Tasks: []subtractionTask{
			{Minuend: 2, Subtrahend: 4},
			{Minuend: 3, Subtrahend: 5},
			{Minuend: 5, Subtrahend: 7},
			{Minuend: 1, Subtrahend: 3},
			{Minuend: 6, Subtrahend: 8},
			{Minuend: 4, Subtrahend: 6},
			{Minuend: 7, Subtrahend: 9},
			{Minuend: 8, Subtrahend: 10},
		},
	},
	{
		Title: "3. Jetzt gemischt",
		Hint:  "Nutze den Zahlenstrahl, wenn du Hilfe brauchst.",
		Tasks: []subtractionTask{
			{Minuend: 3, Subtrahend: 6},
			{Minuend: 6, Subtrahend: 10},
			{Minuend: 2, Subtrahend: 7},
			{Minuend: 5, Subtrahend: 11},
			{Minuend: 1, Subtrahend: 8},
			{Minuend: 4, Subtrahend: 12},
			{Minuend: 3, Subtrahend: 13},
			{Minuend: 2, Subtrahend: 14},
		},
	},
}
