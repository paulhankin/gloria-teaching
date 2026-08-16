package venndiagrams

// Tasks, diagram data and solutions of this worksheet.
// The pupil-facing text stays in German on purpose.

var tasks = []task{
	{
		Key:   "a1",
		Title: "Aufgabe 1",
		Story: "In der Klasse 4b spielen einige Kinder <b>Fu&szlig;ball</b> &#9917; und einige gehen zum <b>Tanzen</b> &#128131;. Manche Kinder machen sogar beides! Im Diagramm siehst du alle Zahlen.",
		Questions: []string{
			"Wie viele Kinder spielen <b>nur</b> Fu&szlig;ball? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>beides</b>? <span class=\"blank\"></span>",
			"Wie viele Kinder spielen <b>insgesamt</b> Fu&szlig;ball? <span class=\"blank\"></span>",
			"Wie viele Kinder sind es <b>zusammen</b>? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "2",
			Labels: []string{"⚽ Fußball", "💃 Tanzen"},
			Colors: []string{"pink", "blue"},
			Values: map[string]string{"A": "12", "AB": "5", "B": "8"},
		},
		Solution: "<h3>Aufgabe 1</h3>\n    <p>a) <b>12</b> &nbsp;&nbsp; b) <b>5</b> &nbsp;&nbsp; c) 12 + 5 = <b>17</b> &nbsp;&nbsp; d) 12 + 5 + 8 = <b>25</b></p>",
	},
	{
		Key:   "a2",
		Title: "Aufgabe 2",
		Story: "In der Musik-AG sind <b>24 M&auml;dchen</b>. <b>15</b> von ihnen spielen <b>Hockey</b> &#127954;, <b>12</b> spielen <b>Gitarre</b> &#127928;, und <b>6</b> M&auml;dchen machen <b>beides</b>.",
		Questions: []string{
			"Wie viele spielen <b>nur</b> Hockey? <span class=\"blank\"></span>",
			"Wie viele spielen <b>nur</b> Gitarre? <span class=\"blank\"></span>",
			"Wie viele M&auml;dchen machen <b>keins</b> von beidem? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "2",
			Box:    "alle 24 Mädchen",
			Labels: []string{"🏑 Hockey", "🎸 Gitarre"},
			Colors: []string{"pink", "blue"},
			Values: map[string]string{"A": "?", "AB": "6", "B": "?", "out": "?"},
		},
		Solution: "<h3>Aufgabe 2</h3>\n    <p class=\"calculation\">nur Hockey = 15 &minus; 6 = 9 &nbsp;|&nbsp; nur Gitarre = 12 &minus; 6 = 6</p>\n    <p>a) <b>9</b> &nbsp;&nbsp; b) <b>6</b> &nbsp;&nbsp; c) 24 &minus; (9 + 6 + 6) = 24 &minus; 21 = <b>3</b></p>",
	},
	{
		Key:   "a3",
		Title: "Aufgabe 3",
		Story: "In einer Umfrage werden <b>30 Kinder</b> gefragt, welche Haustiere sie haben. <b>18</b> Kinder haben einen <b>Hund</b> &#128054;, <b>14</b> Kinder haben eine <b>Katze</b> &#128049;. <b>5</b> Kinder haben <b>weder</b> Hund <b>noch</b> Katze.",
		Questions: []string{
			"Wie viele Kinder haben ein Haustier (Hund <b>oder</b> Katze)? <span class=\"blank\"></span>",
			"Wie viele Kinder haben <b>Hund und Katze</b>? <span class=\"blank\"></span>",
			"Wie viele Kinder haben <b>nur</b> einen Hund? <span class=\"blank\"></span>",
			"Wie viele Kinder haben <b>nur</b> eine Katze? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "2",
			Box:    "alle 30 Kinder",
			Labels: []string{"🐶 Hund", "🐱 Katze"},
			Colors: []string{"pink", "blue"},
			Values: map[string]string{"A": "?", "AB": "?", "B": "?", "out": "5"},
		},
		Solution: "<h3>Aufgabe 3</h3>\n    <p>a) 30 &minus; 5 = <b>25</b> Kinder haben ein Haustier.</p>\n    <p class=\"calculation\">18 + 14 = 32, aber es sind nur 25 &rarr; doppelt gez&auml;hlt: 32 &minus; 25 = 7</p>\n    <p>b) <b>7</b> Kinder haben Hund und Katze &nbsp;&nbsp; c) 18 &minus; 7 = <b>11</b> &nbsp;&nbsp; d) 14 &minus; 7 = <b>7</b></p>\n    <p>Probe: 11 + 7 + 7 + 5 = 30 &#10004;</p>",
	},
	{
		Key:   "a4",
		Title: "Aufgabe 4",
		Story: "Im Schwimmbad sind an einem Nachmittag <b>28 Kinder</b>. <b>16</b> Kinder fahren die <b>Rutsche</b> &#127754; hinunter, <b>19</b> Kinder springen vom <b>Sprungturm</b> &#127946;. <b>4</b> Kinder machen <b>weder</b> das eine <b>noch</b> das andere.",
		Questions: []string{
			"Wie viele Kinder rutschen <b>oder</b> springen? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>beides</b>? <span class=\"blank\"></span>",
			"Wie viele Kinder rutschen <b>nur</b>? <span class=\"blank\"></span>",
			"Wie viele Kinder springen <b>nur</b>? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "2",
			Box:    "alle 28 Kinder",
			Labels: []string{"🌊 Rutsche", "🏊 Sprungturm"},
			Colors: []string{"pink", "yellow"},
			Values: map[string]string{"A": "?", "AB": "?", "B": "?", "out": "4"},
		},
		Solution: "<h3>Aufgabe 4</h3>\n    <p>a) 28 &minus; 4 = <b>24</b> Kinder rutschen oder springen.</p>\n    <p class=\"calculation\">16 + 19 = 35, aber es sind nur 24 &rarr; doppelt gez&auml;hlt: 35 &minus; 24 = 11</p>\n    <p>b) <b>11</b> &nbsp;&nbsp; c) 16 &minus; 11 = <b>5</b> &nbsp;&nbsp; d) 19 &minus; 11 = <b>8</b></p>\n    <p>Probe: 5 + 11 + 8 + 4 = 28 &#10004;</p>",
	},
	{
		Key:   "a5",
		Title: "Aufgabe 5",
		Story: "Im Ferienkurs kann man <b>Schwimmen</b> &#127946;, <b>Reiten</b> &#128014; und <b>Klavier</b> &#127929; w&auml;hlen. Alle Zahlen stehen schon im Diagramm.",
		Questions: []string{
			"Wie viele Kinder <b>schwimmen</b> (insgesamt)? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>genau zwei</b> Sachen? <span class=\"blank\"></span>",
			"Wie viele Kinder reiten, spielen aber <b>kein</b> Klavier? <span class=\"blank\"></span>",
			"Wie viele Kinder sind es <b>insgesamt</b>? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "3",
			Box:    "alle Kinder",
			Labels: []string{"🏊 Schwimmen", "🐎 Reiten", "🎹 Klavier"},
			Colors: []string{"pink", "blue", "yellow"},
			Values: map[string]string{"A": "6", "B": "4", "C": "3", "AB": "2", "AC": "5", "BC": "1", "ABC": "2", "out": "4"},
		},
		Solution: "<h3>Aufgabe 5</h3>\n    <p>a) 6 + 2 + 5 + 2 = <b>15</b></p>\n    <p>b) 2 + 5 + 1 = <b>8</b></p>\n    <p>c) Reiten ohne Klavier: 4 + 2 = <b>6</b></p>\n    <p>d) 6 + 4 + 3 + 2 + 5 + 1 + 2 + 4 = <b>27</b></p>",
	},
	{
		Key:   "a6",
		Title: "Aufgabe 6",
		Story: "Auf dem Sommerfest dürfen <b>40 Kinder</b> Eis probieren.<br><b>22</b> mochten Schokolade &#127851;, <b>18</b> Vanille &#127846;, <b>15</b> Erdbeere &#127827;.<br><b>8</b> mochten Schoko <b>und</b> Vanille, <b>7</b> Schoko <b>und</b> Erdbeere, <b>6</b> Vanille <b>und</b> Erdbeere.<br><b>3</b> Kinder mochten <b>alle drei</b> Sorten.",
		Questions: []string{
			"F&uuml;lle zuerst das ganze Diagramm aus. Beginne <b>in der Mitte</b>!",
			"Wie viele Kinder mochten <b>nur</b> Schokolade? <span class=\"blank\"></span>",
			"Wie viele mochten <b>nur</b> Erdbeere? <span class=\"blank\"></span>",
			"Wie viele mochten <b>genau eine</b> Sorte? <span class=\"blank\"></span>",
			"Wie viele mochten <b>keine</b> der drei Sorten? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "3",
			Box:    "alle 40 Kinder",
			Labels: []string{"🍫 Schokolade", "🍦 Vanille", "🍓 Erdbeere"},
			Colors: []string{"pink", "blue", "yellow"},
			Values: map[string]string{"A": "?", "B": "?", "C": "?", "AB": "?", "AC": "?", "BC": "?", "ABC": "3", "out": "?"},
		},
		Solution: "<h3>Aufgabe 6</h3>\n    <p>Mitte: <b>3</b>. Dann die Zweier-Felder:</p>\n    <p class=\"calculation\">Scho&amp;Van nur: 8 &minus; 3 = 5 &nbsp;|&nbsp; Scho&amp;Erd nur: 7 &minus; 3 = 4 &nbsp;|&nbsp; Van&amp;Erd nur: 6 &minus; 3 = 3</p>\n    <p class=\"calculation\">nur Schoko: 22 &minus; 5 &minus; 4 &minus; 3 = 10 &nbsp;|&nbsp; nur Vanille: 18 &minus; 5 &minus; 3 &minus; 3 = 7 &nbsp;|&nbsp; nur Erdbeere: 15 &minus; 4 &minus; 3 &minus; 3 = 5</p>\n    <p>b) <b>10</b> &nbsp;&nbsp; c) <b>5</b> &nbsp;&nbsp; d) 10 + 7 + 5 = <b>22</b> &nbsp;&nbsp; e) 40 &minus; 37 = <b>3</b></p>\n    <p>Probe: 10 + 7 + 5 + 5 + 4 + 3 + 3 = 37 Kinder mit Eis, dazu 3 ohne = 40 &#10004;</p>",
	},
	{
		Key:   "a7",
		Title: "Aufgabe 7",
		Story: "Im Zeltlager sind <b>36 Kinder</b>.<br><b>20</b> machen Bogenschießen &#127993;, <b>17</b> fahren Kanu &#128758;, <b>15</b> klettern &#129495;.<br><b>4</b> Kinder machen <b>alle drei</b> Sachen.<br><b>10</b> Kinder machen <b>genau zwei</b> Sachen.",
		Questions: []string{
			"Wie viele Kinder machen <b>mindestens eine</b> Aktivit&auml;t? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>gar nichts</b> davon? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>genau eine</b> Aktivit&auml;t? <span class=\"blank\"></span>",
			"Wie viele Kinder machen <b>mindestens zwei</b> Aktivit&auml;ten? <span class=\"blank\"></span>",
		},
		Diagram: diagram{
			Kind:   "3",
			Box:    "alle 36 Kinder",
			Labels: []string{"🏹 Bogen", "🛶 Kanu", "🧗 Klettern"},
			Colors: []string{"pink", "blue", "yellow"},
			Values: map[string]string{"A": "?", "B": "?", "C": "?", "AB": "?", "AC": "?", "BC": "?", "ABC": "4", "out": "?"},
		},
		Solution: "<h3>Aufgabe 7</h3>\n    <p class=\"calculation\">20 + 17 + 15 = 52 (Kinder mehrfach gez&auml;hlt)</p>\n    <p>Die 10 Kinder mit genau zwei Aktivit&auml;ten wurden 1-mal zu viel gez&auml;hlt (&minus;10).<br>\n       Die 4 Kinder mit drei Aktivit&auml;ten wurden 2-mal zu viel gez&auml;hlt (&minus;8).</p>\n    <p class=\"calculation\">52 &minus; 10 &minus; 8 = 34</p>\n    <p>a) <b>34</b> &nbsp;&nbsp; b) 36 &minus; 34 = <b>2</b> &nbsp;&nbsp; c) 34 &minus; 10 &minus; 4 = <b>20</b> &nbsp;&nbsp; d) 10 + 4 = <b>14</b></p>",
	},
}
