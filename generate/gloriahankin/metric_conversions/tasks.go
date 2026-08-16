package metricconversions

type exercise struct {
	Prompt string
	Answer string
}

type comparison struct {
	Left   string
	Right  string
	Sign   string
	Reason string
}

type ordering struct {
	Values string
	Answer string
}

type wordProblem struct {
	Title       string
	Story       string
	Question    string
	Answer      string
	Calculation string
}

// Worksheet tasks. Pupil-facing text stays German on purpose.
var directConversions = []exercise{
	{Prompt: `7 cm = <span class="answer-blank"></span> mm`, Answer: "7 cm = 70 mm"},
	{Prompt: `90 mm = <span class="answer-blank"></span> cm`, Answer: "90 mm = 9 cm"},
	{Prompt: `6 dm = <span class="answer-blank"></span> cm`, Answer: "6 dm = 60 cm"},
	{Prompt: `340 cm = <span class="answer-blank"></span> dm`, Answer: "340 cm = 34 dm"},
	{Prompt: `8 m = <span class="answer-blank"></span> dm`, Answer: "8 m = 80 dm"},
	{Prompt: `120 dm = <span class="answer-blank"></span> m`, Answer: "120 dm = 12 m"},
	{Prompt: `5 m = <span class="answer-blank"></span> cm`, Answer: "5 m = 500 cm"},
	{Prompt: `2300 mm = <span class="answer-blank"></span> cm`, Answer: "2300 mm = 230 cm"},
	{Prompt: `4 km = <span class="answer-blank"></span> m`, Answer: "4 km = 4000 m"},
	{Prompt: `7000 m = <span class="answer-blank"></span> km`, Answer: "7000 m = 7 km"},
	{Prompt: `3 m = <span class="answer-blank"></span> mm`, Answer: "3 m = 3000 mm"},
	{Prompt: `2 km = <span class="answer-blank wide"></span> cm`, Answer: "2 km = 200 000 cm"},
}

var composedConversions = []exercise{
	{Prompt: `4 cm 7 mm = <span class="answer-blank"></span> mm`, Answer: "4 cm 7 mm = 47 mm"},
	{Prompt: `126 mm = <span class="answer-blank"></span> cm <span class="answer-blank short"></span> mm`, Answer: "126 mm = 12 cm 6 mm"},
	{Prompt: `3 dm 8 cm = <span class="answer-blank"></span> cm`, Answer: "3 dm 8 cm = 38 cm"},
	{Prompt: `74 cm = <span class="answer-blank"></span> dm <span class="answer-blank short"></span> cm`, Answer: "74 cm = 7 dm 4 cm"},
	{Prompt: `5 m 6 dm = <span class="answer-blank"></span> dm`, Answer: "5 m 6 dm = 56 dm"},
	{Prompt: `93 dm = <span class="answer-blank"></span> m <span class="answer-blank short"></span> dm`, Answer: "93 dm = 9 m 3 dm"},
	{Prompt: `2 m 35 cm = <span class="answer-blank"></span> cm`, Answer: "2 m 35 cm = 235 cm"},
	{Prompt: `468 cm = <span class="answer-blank"></span> m <span class="answer-blank short"></span> cm`, Answer: "468 cm = 4 m 68 cm"},
	{Prompt: `3 km 250 m = <span class="answer-blank wide"></span> m`, Answer: "3 km 250 m = 3250 m"},
	{Prompt: `4800 m = <span class="answer-blank"></span> km <span class="answer-blank"></span> m`, Answer: "4800 m = 4 km 800 m"},
	{Prompt: `6 m 4 cm = <span class="answer-blank wide"></span> cm`, Answer: "6 m 4 cm = 604 cm"},
	{Prompt: `7025 m = <span class="answer-blank"></span> km <span class="answer-blank"></span> m`, Answer: "7025 m = 7 km 25 m"},
}

var comparisons = []comparison{
	{Left: "45 cm", Right: "5 dm", Sign: "<", Reason: "45 cm < 50 cm"},
	{Left: "1200 mm", Right: "1 m 20 cm", Sign: "=", Reason: "1200 mm = 1200 mm"},
	{Left: "3 m 8 dm", Right: "390 cm", Sign: "<", Reason: "380 cm < 390 cm"},
	{Left: "2 km", Right: "1999 m", Sign: ">", Reason: "2000 m > 1999 m"},
	{Left: "75 dm", Right: "7 m 50 cm", Sign: "=", Reason: "750 cm = 750 cm"},
	{Left: "640 cm", Right: "6 m 4 dm", Sign: "=", Reason: "640 cm = 640 cm"},
	{Left: "9 cm 8 mm", Right: "100 mm", Sign: "<", Reason: "98 mm < 100 mm"},
	{Left: "1 km 50 m", Right: "10 500 dm", Sign: "=", Reason: "1050 m = 1050 m"},
}

var orderings = []ordering{
	{Values: "8 dm &lt; 90 cm &lt; 950 mm &lt; 1 m", Answer: "8 dm < 90 cm < 950 mm < 1 m"},
	{Values: "1750 m &lt; 1 km 900 m &lt; 2 km &lt; 21 000 dm", Answer: "1750 m < 1 km 900 m < 2 km < 21 000 dm"},
	{Values: "390 cm &lt; 3999 mm &lt; 4 m 5 cm &lt; 41 dm", Answer: "390 cm < 3999 mm < 4 m 5 cm < 41 dm"},
}

var wordProblems = []wordProblem{
	{
		Title:    "Ameisenweg",
		Story:    "Eine Ameise läuft zuerst <b>85 cm</b> und danach noch <b>430 mm</b>.",
		Question: "Wie viele Zentimeter läuft sie insgesamt?",
		Answer:   "Die Ameise läuft insgesamt <b>128 cm</b>.",
		Calculation: "430 mm = 43 cm<br>" +
			"85 cm + 43 cm = 128 cm",
	},
	{
		Title:    "Geschenkband",
		Story:    "Ein Geschenkband ist <b>3 m</b> lang. Mia schneidet <b>85 cm</b> und <b>1 m 20 cm</b> ab.",
		Question: "Wie viel Band bleibt übrig?",
		Answer:   "Es bleiben <b>95 cm</b> Geschenkband übrig.",
		Calculation: "3 m = 300 cm; 1 m 20 cm = 120 cm<br>" +
			"300 cm − 85 cm − 120 cm = 95 cm",
	},
	{
		Title:    "Wanderung",
		Story:    "Am Vormittag wandert eine Klasse <b>2 km 350 m</b>, am Nachmittag <b>1750 m</b>.",
		Question: "Wie lang ist die ganze Wanderung?",
		Answer:   "Die Wanderung ist <b>4 km 100 m</b> lang.",
		Calculation: "2 km 350 m = 2350 m<br>" +
			"2350 m + 1750 m = 4100 m = 4 km 100 m",
	},
	{
		Title:    "Stifte vergleichen",
		Story:    "Ein Farbstift ist <b>18 cm 5 mm</b> lang. Ein Filzstift ist <b>142 mm</b> lang.",
		Question: "Um wie viele Millimeter ist der Farbstift länger?",
		Answer:   "Der Farbstift ist <b>43 mm</b> länger.",
		Calculation: "18 cm 5 mm = 185 mm<br>" +
			"185 mm − 142 mm = 43 mm",
	},
}
