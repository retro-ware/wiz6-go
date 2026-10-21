package game

// dtLog is set only by the differential tests (difftest_test.go): it records random
// numbers, messages, numbers and sounds, and turns off delays and drawing that the
// original's test harness stubs. Nil in the game.
var dtLog func(kind string, v ...int)
