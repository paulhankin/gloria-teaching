package halfhourtimes

type clockTime struct {
	Hour   int `json:"hour"`
	Minute int `json:"minute"`
}

type clockTask struct {
	Key  string
	Time clockTime
}

type drawingTask struct {
	Key    string
	Prompt string
	Time   clockTime
}

type laterTask struct {
	Key   string
	Start clockTime
}

// Worksheet tasks. Pupil-facing text stays German on purpose.
var readingTasks = []clockTask{
	{Key: "read-1", Time: clockTime{Hour: 1, Minute: 30}},
	{Key: "read-2", Time: clockTime{Hour: 4, Minute: 30}},
	{Key: "read-3", Time: clockTime{Hour: 7, Minute: 30}},
	{Key: "read-4", Time: clockTime{Hour: 10, Minute: 30}},
	{Key: "read-5", Time: clockTime{Hour: 2, Minute: 30}},
	{Key: "read-6", Time: clockTime{Hour: 5, Minute: 30}},
	{Key: "read-7", Time: clockTime{Hour: 8, Minute: 30}},
	{Key: "read-8", Time: clockTime{Hour: 11, Minute: 30}},
}

var drawingTasks = []drawingTask{
	{Key: "draw-1", Prompt: "halb eins", Time: clockTime{Hour: 12, Minute: 30}},
	{Key: "draw-2", Prompt: "halb vier", Time: clockTime{Hour: 3, Minute: 30}},
	{Key: "draw-3", Prompt: "halb sieben", Time: clockTime{Hour: 6, Minute: 30}},
	{Key: "draw-4", Prompt: "halb zehn", Time: clockTime{Hour: 9, Minute: 30}},
	{Key: "draw-5", Prompt: "2:30 Uhr", Time: clockTime{Hour: 2, Minute: 30}},
	{Key: "draw-6", Prompt: "5:30 Uhr", Time: clockTime{Hour: 5, Minute: 30}},
	{Key: "draw-7", Prompt: "8:30 Uhr", Time: clockTime{Hour: 8, Minute: 30}},
	{Key: "draw-8", Prompt: "11:30 Uhr", Time: clockTime{Hour: 11, Minute: 30}},
}

var laterTasks = []laterTask{
	{Key: "later-1", Start: clockTime{Hour: 8, Minute: 0}},
	{Key: "later-2", Start: clockTime{Hour: 1, Minute: 30}},
	{Key: "later-3", Start: clockTime{Hour: 10, Minute: 0}},
	{Key: "later-4", Start: clockTime{Hour: 4, Minute: 30}},
	{Key: "later-5", Start: clockTime{Hour: 12, Minute: 0}},
	{Key: "later-6", Start: clockTime{Hour: 6, Minute: 30}},
	{Key: "later-7", Start: clockTime{Hour: 3, Minute: 0}},
	{Key: "later-8", Start: clockTime{Hour: 9, Minute: 30}},
}
