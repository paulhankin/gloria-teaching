package leastcommonmultiple

type sequenceTask struct {
	Start   string
	Missing string
	Answer  string
}

type multiplePair struct {
	A      int
	B      int
	Answer int
}

type lcmTask struct {
	A      int
	B      int
	Answer int
}

type wordProblem struct {
	Title       string
	Story       string
	Question    string
	Calculation string
	Answer      string
}

// Worksheet tasks. Pupil-facing text stays German on purpose.
var sequences = []sequenceTask{
	{Start: "4, 8, 12,", Missing: "__, __, __, __", Answer: "4, 8, 12, 16, 20, 24, 28"},
	{Start: "6, 12, 18,", Missing: "__, __, __, __", Answer: "6, 12, 18, 24, 30, 36, 42"},
	{Start: "7, 14, 21,", Missing: "__, __, __, __", Answer: "7, 14, 21, 28, 35, 42, 49"},
	{Start: "9, 18, 27,", Missing: "__, __, __, __", Answer: "9, 18, 27, 36, 45, 54, 63"},
}

var listingTasks = []multiplePair{
	{A: 2, B: 3, Answer: 6},
	{A: 3, B: 4, Answer: 12},
	{A: 4, B: 6, Answer: 12},
	{A: 5, B: 7, Answer: 35},
}

var practiceTasks = []lcmTask{
	{A: 2, B: 5, Answer: 10},
	{A: 3, B: 6, Answer: 6},
	{A: 4, B: 5, Answer: 20},
	{A: 4, B: 8, Answer: 8},
	{A: 6, B: 8, Answer: 24},
	{A: 7, B: 3, Answer: 21},
	{A: 9, B: 6, Answer: 18},
	{A: 8, B: 10, Answer: 40},
	{A: 12, B: 4, Answer: 12},
	{A: 5, B: 6, Answer: 30},
	{A: 7, B: 8, Answer: 56},
	{A: 9, B: 12, Answer: 36},
}

var wordProblems = []wordProblem{
	{
		Title:    "Zwei Glocken",
		Story:    "Eine kleine Glocke läutet alle <b>4 Minuten</b>, eine grosse alle <b>6 Minuten</b>. Jetzt läuten beide gleichzeitig.",
		Question: "Nach wie vielen Minuten läuten sie wieder zusammen?",
		Calculation: "V(4): 4, 8, <b>12</b>, …<br>" +
			"V(6): 6, <b>12</b>, …<br>kgV(4, 6) = 12",
		Answer: "Nach <b>12 Minuten</b> läuten beide wieder zusammen.",
	},
	{
		Title:    "Training",
		Story:    "Lina geht jeden <b>3. Tag</b> schwimmen. Noah geht jeden <b>5. Tag</b> schwimmen. Heute trainieren sie zusammen.",
		Question: "In wie vielen Tagen treffen sie sich beim Training wieder?",
		Calculation: "V(3): 3, 6, 9, 12, <b>15</b>, …<br>" +
			"V(5): 5, 10, <b>15</b>, …<br>kgV(3, 5) = 15",
		Answer: "Sie treffen sich in <b>15 Tagen</b> wieder.",
	},
	{
		Title:    "Blinklichter",
		Story:    "Ein blaues Licht blinkt alle <b>8 Sekunden</b>, ein gelbes alle <b>12 Sekunden</b>. Gerade blinken beide zusammen.",
		Question: "Nach wie vielen Sekunden blinken beide wieder gleichzeitig?",
		Calculation: "V(8): 8, 16, <b>24</b>, …<br>" +
			"V(12): 12, <b>24</b>, …<br>kgV(8, 12) = 24",
		Answer: "Nach <b>24 Sekunden</b> blinken beide wieder zusammen.",
	},
	{
		Title:    "Perlenmuster",
		Story:    "Auf einer Schnur beginnt alle <b>6 Perlen</b> ein rotes Muster und alle <b>9 Perlen</b> ein blaues Muster. Bei Perle 0 beginnen beide.",
		Question: "Bei welcher Perle beginnen beide Muster wieder zusammen?",
		Calculation: "V(6): 6, 12, <b>18</b>, …<br>" +
			"V(9): 9, <b>18</b>, …<br>kgV(6, 9) = 18",
		Answer: "Bei der <b>18. Perle</b> beginnen beide Muster wieder zusammen.",
	},
}
