package answerchecks

type arithmeticTask struct {
	Operation string
	Top       int
	Bottom    int
	Claim     int
	Correct   int
}

type priceTask struct {
	Key       string
	Kind      string
	Title     string
	Story     string
	Question  string
	Names     [2]string
	Values    [2]int
	CheckText string
}

type logicTask struct {
	Key        string
	Kind       string
	Title      string
	Story      string
	Question   string
	Answer     string
	Reasoning  string
	CheckItems []string
	Labels     []string
}

// Worksheet tasks. Pupil-facing text stays German on purpose.
var arithmeticTasks = []arithmeticTask{
	{Operation: "−", Top: 764, Bottom: 328, Claim: 436, Correct: 436},
	{Operation: "−", Top: 902, Bottom: 457, Claim: 455, Correct: 445},
	{Operation: "+", Top: 368, Bottom: 247, Claim: 615, Correct: 615},
	{Operation: "+", Top: 586, Bottom: 179, Claim: 755, Correct: 765},
}

var priceTasks = []priceTask{
	{
		Key: "price-fruit", Kind: "fruit", Title: "Apfel und Birne",
		Story: "Ein Apfelkorb und ein Birnenkorb kosten zusammen <b>20 Franken</b>. " +
			"Der Birnenkorb kostet <b>4 Franken mehr</b> als der Apfelkorb.",
		Question:  "Wie viel kostet jeder Korb?",
		Names:     [2]string{"Apfelkorb", "Birnenkorb"},
		Values:    [2]int{8, 12},
		CheckText: "8 + 12 = 20 Franken",
	},
	{
		Key: "price-books", Kind: "books", Title: "Buch und Heft",
		Story: "Ein Sachbuch und ein Forscherheft kosten zusammen <b>28 Franken</b>. " +
			"Das Sachbuch kostet <b>dreimal so viel</b> wie das Forscherheft.",
		Question:  "Wie viel kosten Buch und Heft?",
		Names:     [2]string{"Sachbuch", "Forscherheft"},
		Values:    [2]int{21, 7},
		CheckText: "21 + 7 = 28 Franken",
	},
}

var logicTasks = []logicTask{
	{
		Key: "logic-ages", Kind: "house", Title: "Die Kinder im Haus",
		Story: "Die fünf Kinder in unserem Haus heissen Sarah, Vito, Lena, Antonio und Henry. " +
			"Sie sind <b>4, 5, 6, 7 und 8 Jahre</b> alt. Jedes Alter kommt genau einmal vor. " +
			"Vito ist der Jüngste. Lena ist 2 Jahre älter als Antonio und 1 Jahr jünger als Henry.",
		Question:  "Wie alt ist Sarah?",
		Answer:    "Sarah ist <b>6 Jahre</b> alt.",
		Reasoning: "Vito = 4. Für Antonio, Lena und Henry passen nur 5, 7 und 8. Damit bleibt für Sarah die 6.",
		CheckItems: []string{
			"Vito ist mit 4 Jahren der Jüngste.",
			"Lena (7) ist 2 Jahre älter als Antonio (5).",
			"Lena (7) ist 1 Jahr jünger als Henry (8).",
			"Die Alter 4, 5, 6, 7 und 8 kommen je einmal vor.",
		},
		Labels: []string{"Sarah", "Vito", "Lena", "Antonio", "Henry"},
	},
	{
		Key: "logic-race", Kind: "race", Title: "Das Seifenkistenrennen",
		Story: "Lea, Noah, Amir, Mia und Zoe belegen die Plätze <b>1 bis 5</b>. Jedes Kind hat einen anderen Platz. " +
			"Lea wird Vierte. Zoe kommt genau 2 Plätze hinter Amir ins Ziel. Noah ist direkt vor Mia.",
		Question:  "Welchen Platz belegt Mia?",
		Answer:    "Mia belegt den <b>2. Platz</b>.",
		Reasoning: "Amir kann nur Dritter und Zoe Fünfte sein. Dann bleiben für Noah und Mia die Plätze 1 und 2; direkt nacheinander sind das Noah auf Platz 1 und Mia auf Platz 2.",
		CheckItems: []string{
			"Richtige Reihenfolge: Noah (1.), Mia (2.), Amir (3.), Lea (4.), Zoe (5.).",
			"Alle fünf Plätze kommen genau einmal vor.",
			"Zoe (5.) liegt genau 2 Plätze hinter Amir (3.).",
			"Noah (1.) ist direkt vor Mia (2.), Lea ist 4.",
		},
		Labels: []string{"Lea", "Noah", "Amir", "Mia", "Zoe"},
	},
	{
		Key: "logic-stamps", Kind: "stamps", Title: "Die Briefmarkenalben",
		Story: "In fünf kleinen Alben sind <b>2, 4, 6, 8 und 10 Briefmarken</b>. Jedes Album enthält eine andere Anzahl. " +
			"Das blaue Album enthält am wenigsten. Im grünen Album sind 4 Marken mehr als im roten. " +
			"Im grünen Album sind 2 Marken weniger als im gelben.",
		Question:  "Wie viele Briefmarken sind im orangen Album?",
		Answer:    "Im orangen Album sind <b>6 Briefmarken</b>.",
		Reasoning: "Blau = 2. Für Rot, Grün und Gelb passen nur 4, 8 und 10. Damit bleibt die 6 für Orange.",
		CheckItems: []string{
			"Blau (2) enthält am wenigsten.",
			"Grün (8) hat 4 Marken mehr als Rot (4).",
			"Grün (8) hat 2 Marken weniger als Gelb (10).",
			"Die Anzahlen 2, 4, 6, 8 und 10 kommen je einmal vor.",
		},
		Labels: []string{"Rot", "Blau", "Grün", "Gelb", "Orange"},
	},
}
