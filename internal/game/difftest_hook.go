package game

// dtLog is set only by the differential tests (difftest_test.go): it records random
// numbers, messages, numbers and sounds, and turns off delays and drawing that the
// original's test harness stubs. Nil in the game.
var dtLog func(kind string, v ...int)

// dtKeys is the difftest's scripted input: keyAvailable is whether one is left, getKey
// takes the next (the original's stubs do the same).
var dtKeys []int

// dtTexts: getMessage gives the real text in the difftest (else an empty one).
var dtTexts bool
