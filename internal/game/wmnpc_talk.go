package game

import (
	"bytes"
	"strings"
)

// WMNPC talk: the player types a line, the parser sorts its words (verbs, question words,
// pronouns, the NPC's topics...) and the NPC answers from the topic's messages.
//
// A topic word is stored with a mark that gives its plural: "word%" adds S, "word@" is the
// same, "word^nsuffix" drops n letters and adds suffix. Its first message character - 32
// is a delta: a negative one makes it an alias of an earlier topic. The NPC's own topics
// (from message Unk68[0], step 5) come first, then the common ones (8000, step 5); topic
// k's answers are messages base+1..base+4.

// Word lists, by fbf7f's list number.
const (
	npcListVerb    = 1 // 7000, 10
	npcListFiller  = 2 // 7140, 7
	npcListInterj  = 3 // 7120, 10
	npcListSubject = 4 // 7050, 7
	npcListPerson  = 7 // 7070, 5 pairs: "you" answered as "me"
	npcListAsk     = 8 // 7030, 7: who, what, where...
	npcListPronoun = 9 // 7090, 11
)

var (
	npcWords      [10][]string // the word lists
	npcPairs      [5][2]string // person words and their answer form
	npcWord6      string       // 0x6a84 (7040)
	npcWord7      string       // 0x6a8e (7041)
	npcSynonyms   []string     // 0x6316: "canon/syn/syn/"
	npcVocab      []string     // 0x58da
	npcVocabRef   []int        // g58d8: the delta
	npcVocabOwn   int          // g6a9e: the NPC's own topics come first
	npcInterj     [2]int       // g6a98, g6a9a: the answers to interjections 0 and 1
	npcTalkMember int          // g6a9c
	tk            npcTalkState
)

// npcTalkState is what the parser gathered from the words so far (0x5894..0x58d6).
type npcTalkState struct {
	topic, plural           int // g5894, g5896
	verb                    int // g5898
	interj                  int // g589a
	lastTopic, lastPlural   int // g589c, g589e: the line before
	lastVerb, lastAsk       int // g58a0, g58a2
	ask                     [7]bool
	negated                 bool // g58b2
	question                bool // g58b4
	aboutSubject            bool // g58b6
	done                    bool // g58c2: the talk ends
	subject, object, person int  // g58ca, g58cc, g58ce
	topic2, plural2         int  // g58d0, g58d2: a second topic
	verb2                   int  // g58d4
	questionMark            bool // g58d6
	prevClass               int  // g6a82: the class of the word before
}

// npcLoadVocabulary (WMNPC d46f) loads the words the NPC knows (d2d3, d392, d3f3, d431).
func npcLoadVocabulary() {
	npcInterj = [2]int{int(npcRec.Unk68[2]), int(npcRec.Unk68[3])}
	npcVocab, npcVocabRef = npcVocab[:0], npcVocabRef[:0]
	for i, base := range []int{int(npcRec.Unk68[0]), 8000} {
		for msg := base; ; msg += 5 {
			s := getMessage(msg)
			if msgNotFound {
				break
			}
			ref := -32
			if len(s) > 0 {
				ref += int(s[0])
				s = s[1:]
			}
			npcVocab, npcVocabRef = append(npcVocab, s), append(npcVocabRef, ref)
		}
		if i == 0 {
			npcVocabOwn = len(npcVocab)
		}
	}
	for _, l := range []struct{ list, msg, n int }{
		{npcListVerb, 7000, 10}, {npcListAsk, 7030, 7}, {npcListSubject, 7050, 7},
		{npcListPronoun, 7090, 11}, {npcListInterj, 7120, 10}, {npcListFiller, 7140, 7},
	} {
		npcWords[l.list] = npcWords[l.list][:0]
		for i := range l.n {
			npcWords[l.list] = append(npcWords[l.list], getMessage(l.msg+i))
		}
	}
	npcWord6, npcWord7 = getMessage(7040), getMessage(7041)
	npcWords[npcListPerson] = npcWords[npcListPerson][:0]
	for i := range npcPairs {
		npcPairs[i] = [2]string{getMessage(7070 + 2*i), getMessage(7071 + 2*i)}
		npcWords[npcListPerson] = append(npcWords[npcListPerson], npcPairs[i][0])
	}
	npcSynonyms = npcSynonyms[:0]
	for i := range 38 {
		npcSynonyms = append(npcSynonyms, getMessage(7160+i))
	}
}

// npcTopicMsg (WMNPC cdb0) is the first message of topic k.
func npcTopicMsg(k int) int {
	if k < npcVocabOwn {
		return k*5 + int(npcRec.Unk68[0])
	}
	return (k-npcVocabOwn)*5 + 8000
}

// npcSingular (WMNPC bbfa) is topic k's word.
func npcSingular(k int) string {
	w := npcVocab[k]
	for _, c := range "%@^" { // the C looks for each marker in turn
		if i := strings.IndexRune(w, c); i >= 0 {
			return w[:i]
		}
	}
	return w
}

// npcPlural (WMNPC bcec) is topic k's plural.
func npcPlural(k int) string {
	w := npcVocab[k]
	if i := strings.IndexByte(w, '%'); i >= 0 {
		return w[:i] + "S"
	}
	if i := strings.IndexByte(w, '@'); i >= 0 {
		return w[:i]
	}
	if i := strings.IndexByte(w, '^'); i >= 0 && i+1 < len(w) {
		n := min(int(w[i+1])-'0', i)
		return w[:i-n] + w[i+2:]
	}
	return w
}

// npcTopicWord (WMNPC be7a) is topic k's word, plural if plural.
func npcTopicWord(k, plural int) string {
	if plural != 0 {
		return npcPlural(k)
	}
	return npcSingular(k)
}

// npcBlankWord (WMNPC bb01) marks the first n bytes of s as used (x), so that a phrase of
// several words reads as one.
func npcBlankWord(s []byte, n int) {
	for i := 0; i < n && i < len(s); i++ {
		s[i] = 'x'
	}
}

// npcEat blanks w at the start of s if s starts with it; whether it did.
func npcEat(s []byte, w string) bool {
	if !bytes.HasPrefix(s, []byte(w)) {
		return false
	}
	npcBlankWord(s, len(w)-1)
	return true
}

// npcMatchVocab (WMNPC bea1) is the topic s starts with, and whether in the plural; -1.
func npcMatchVocab(s []byte) (int, int) {
	for k := range npcVocab {
		if npcEat(s, npcSingular(k)+" ") {
			return k, 0
		}
		if npcEat(s, npcPlural(k)+" ") {
			return k, 1
		}
	}
	return -1, 0
}

// npcMatchList (WMNPC bf7f) is the word of list s starts with, or -1.
func npcMatchList(s []byte, list int) int {
	for i, w := range npcWords[list] {
		if npcEat(s, w+" ") {
			return i
		}
	}
	return -1
}

// npcSwapPair (WMNPC c115) turns a person word into its answer form ("you" into "me").
func npcSwapPair(s string) string {
	for _, p := range npcPairs {
		if s == p[0] {
			return p[1]
		}
	}
	return s
}

// npcAnswerWord is topic k's word as the NPC says it.
func npcAnswerWord(k, plural int) string { return npcSwapPair(npcTopicWord(k, plural)) }

// npcRumor (WMNPC c16d) is one of the NPC's rumours (message Unk68[4]: two count bytes,
// then "rumour/rumour/..."): mode 0 one of the first ones, 1 one of the rest. The rumour
// also becomes the topic.
func npcRumor(mode int) string {
	s := getMessage(int(npcRec.Unk68[4]))
	if len(s) < 2 {
		return ""
	}
	first, rest := int(s[0]), int(s[1])
	parts := strings.Split(s[2:], "/")
	n := len(parts) - 1
	var k int
	if mode == 0 {
		k = random(min(n, first))
	} else {
		k = random(n-rest) + rest
	}
	r := parts[k]
	tk.topic, tk.plural = npcMatchVocab([]byte(r + " "))
	return r
}

// npcWordEnd (WMNPC c282) is the index after the word of s at i, or -1 if none is there.
func npcWordEnd(s []byte, i int) int {
	j := i
	for j < len(s) && s[j] != 0 && s[j] != ' ' {
		j++
	}
	if j == i {
		return -1
	}
	return j
}

// npcSynonym (WMNPC c2bf) is word w (with its space) or the word it stands for.
func npcSynonym(w string) string {
	for _, e := range npcSynonyms {
		parts := strings.Split(e, "/")
		for _, p := range parts[1:] {
			if p != "" && strings.HasPrefix(w, p+" ") {
				return parts[0] + " "
			}
		}
	}
	return w
}

// npcSynonyms (WMNPC c3eb) replaces each word of s by the word it stands for.
func npcReplaceSynonyms(s []byte) []byte {
	var out []byte
	for i := 0; ; {
		for i < len(s) && s[i] == ' ' {
			i++
		}
		j := npcWordEnd(s, i)
		if j == -1 {
			break
		}
		out = append(out, npcSynonym(string(s[i:j])+" ")...)
		i = j
	}
	return out
}

// npcStripPunct (WMNPC bb9a) turns punctuation into spaces, noting a question mark.
func npcStripPunct(s []byte) {
	tk.questionMark = false
	for i, c := range s {
		switch c {
		case '?':
			tk.questionMark = true
			s[i] = ' '
		case '!', '.', ',':
			s[i] = ' '
		}
	}
}

// npcSetAsk (WMNPC bb28) notes question word k (-1: not a question).
func npcSetAsk(k int) {
	tk.question = k != -1
	if k >= 0 && k < len(tk.ask) {
		tk.ask[k] = true
	}
}

// npcSetTopic takes topic k unless one is set: then k is the second topic.
func npcSetTopic(k, plural int) {
	if tk.topic == -1 || tk.topic == 0 {
		tk.topic, tk.plural = k, plural
	} else if tk.topic != k && k != 0 {
		tk.topic2, tk.plural2 = k, plural
	}
}

// npcParseWord (WMNPC c4c9) sorts the word at the start of s and returns its class.
func npcParseWord(s []byte) int {
	prev := tk.prevClass
	if npcMatchList(s, npcListFiller) != -1 {
		return 2
	}
	if k := npcMatchList(s, npcListInterj); k != -1 {
		tk.interj = k
		if k == 1 {
			tk.done = true
		}
		return 3
	}
	if k := npcMatchList(s, npcListAsk); k != -1 {
		tk.lastAsk = k
		npcSetAsk(k)
		tk.verb, tk.verb2 = -1, -1
		return 8
	}
	if k := npcMatchList(s, npcListSubject); k != -1 {
		if prev == 7 || prev == 0 {
			tk.aboutSubject = true
			tk.subject = tk.person
		}
		if k == 0 {
			tk.negated = true
		}
		return 4
	}
	if npcEat(s, npcWord6) {
		return 5
	}
	if k := npcMatchList(s, npcListVerb); k != -1 {
		if prev == 6 {
			tk.verb = -1
			tk.subject, tk.object = tk.object, tk.subject
		}
		if tk.verb != -1 {
			tk.verb2 = k
		} else {
			tk.verb = k
		}
		if prev == 7 {
			tk.subject = tk.person
		}
		return 1
	}
	if npcEat(s, npcWord7) {
		return 6
	}
	if npcMatchList(s, npcListPronoun) != -1 {
		npcSetTopic(tk.lastTopic, tk.lastPlural)
		return 0
	}
	if k, pl := npcMatchVocab(s); k != -1 {
		switch {
		case prev == 4:
			tk.question = true
			if k == 0 {
				tk.subject = 0
			}
		case (prev == 1 || prev == 6) && k == 0:
			tk.object = 0
		}
		if k == 0 {
			tk.person = 0
		}
		npcSetTopic(k, pl)
		return 0
	}
	if k := npcMatchList(s, npcListPerson); k != -1 {
		switch prev {
		case 4:
			tk.question = true
			tk.subject = k
		case 1, 6:
			tk.object = k
		}
		tk.person = k
		return 7
	}
	if prev == 6 {
		tk.verb = -1
	}
	return 10
}

// npcAnswerSpecial (WMNPC c9d7) answers a line with two topics, no topic, two verbs or no
// verb: -1 if it did, 0 to answer the topic.
func npcAnswerSpecial() int {
	if tk.verb == -1 && tk.negated {
		tk.verb = 1
	}
	if tk.topic2 != -1 {
		npcPrintWrapped(replaceCharWithString(getMessage(1600), npcAnswerWord(tk.topic, tk.plural), '^'), 5)
		npcSay(1601, npcAnswerWord(tk.topic2, tk.plural2))
		return -1
	}
	if tk.interj == -1 && tk.topic == -1 {
		switch {
		case tk.verb != -1:
			npcSay(1602, npcWords[npcListVerb][tk.verb])
		case random(2) == 0:
			npcSay(1603, "")
		default:
			npcSay(1604, "")
		}
		return -1
	}
	if tk.verb2 != -1 {
		if tk.topic != -1 {
			npcSay(1605, npcAnswerWord(tk.topic, 1))
			tk.lastVerb, tk.lastTopic, tk.lastPlural, tk.lastAsk = 1, tk.topic, tk.plural, 1
		} else {
			npcSay(1606, "")
		}
		return -1
	}
	if tk.interj == -1 && tk.verb == -1 {
		if tk.topic == -1 {
			npcSay(1607, "")
			return -1
		}
		tk.question, tk.ask[1], tk.verb = true, true, 1
	}
	return 0
}

// npcAnswerInterjection (WMNPC cb7f) answers an interjection: 2 repeats the last
// question, 6 offends, 7 changes the subject, 9 flatters.
func npcAnswerInterjection() {
	tk.verb, tk.topic, tk.plural = -1, -1, 0
	a := npcAttitude(npcTalkMember)
	switch tk.interj {
	case 0:
		npcSay(npcInterj[0], "")
	case 1:
		if npcInterj[1] > 0 {
			npcSay(npcInterj[1], "")
			npcWait(messageDelay)
		}
	case 2:
		npcSay(1608, "")
		tk.verb, tk.topic, tk.plural = tk.lastVerb, tk.lastTopic, tk.lastPlural
		tk.question = true
		npcSetAsk(tk.lastAsk)
	case 3:
		npcSay(1609, "")
		tk.lastAsk = -1
	case 4, 5, 8:
		npcSay(1606+tk.interj, "")
	case 6:
		npcSay(1612, "")
		*a -= int8(random(5) + 1)
		if random(2) == 1 {
			tk.done = true
		}
	case 7:
		npcSay(1613, "")
		tk.topic, tk.plural, tk.verb = -1, 0, -1
		tk.lastTopic, tk.lastPlural, tk.lastVerb, tk.lastAsk = -1, 0, -1, -1
	case 9:
		npcSay(1615, "")
		if random(2) == 1 && *a < 50 {
			*a += int8(random(3) + 1)
		}
	}
}

// npcAnswerOrScript (WMNPC cd43) says message msg; an empty one says 1616 with message
// fallback, "$n" runs script n instead (ending the talk if it ends the meeting).
func npcAnswerOrScript(msg, fallback int) {
	s := getMessage(msg)
	switch {
	case s == "":
		npcSay(1616, getMessage(fallback))
	case s[0] == '$':
		if len(s) > 1 && npcRunScript(npcTalkMember, int(s[1])) == 1 {
			tk.done = true
		}
	default:
		npcSay(msg, "")
	}
}

// npcAnswer (WMNPC cde1) answers the verb about the topic.
func npcAnswer() {
	k := tk.topic
	if npcVocabRef[k] < 0 {
		k += npcVocabRef[k]
	}
	m := npcTopicMsg(k)
	ask := tk.ask
	none := ask == [7]bool{}
	q := tk.question
	switch tk.verb {
	case 0, 1, 2, 3:
		if tk.verb == 0 {
			if q && k == 0 && (ask[0] || ask[1]) {
				if npcRec.Unk68[4] > 0 {
					npcSay(1617, npcRumor(0))
				} else {
					npcSay(1638, "")
				}
				return
			}
			if !q && !tk.aboutSubject {
				q, tk.question = true, true
				ask[1], tk.ask[1] = true, true
			}
			if q && (ask[6] || ask[1]) {
				npcAnswerOrScript(m+3, 1625)
				return
			}
		}
		if !q {
			if k == 0 {
				npcSay(1628, "")
			} else {
				npcSay(1635, "")
			}
			return
		}
		if none {
			ask[1], tk.ask[1] = true, true
		}
		switch {
		case ask[0]:
			if npcVocabRef[k] != 0 {
				npcSay(1618, npcTopicWord(tk.topic, tk.plural))
			} else {
				npcAnswerOrScript(m+1, 1619)
			}
		case ask[1]:
			npcAnswerOrScript(m+2, 1620)
		case ask[2]:
			npcSay(1621, "")
		case ask[3]:
			npcAnswerOrScript(m+4, 1622)
		case ask[4]:
			npcSay(1623, "")
		case ask[5]:
			npcAnswerOrScript(m+2, 1624)
		case ask[6]:
			if k == 0 {
				npcSay(1626, "")
			} else {
				npcSay(1627, "")
			}
		}
	case 4, 5:
		switch {
		case q && tk.subject == 0 && ask[1]:
			npcSay(1639+tk.verb-4, "")
		case q && tk.subject == 0:
			npcSay(1631, "")
		case q:
			npcSay(1627, "")
		case tk.verb == 4 && k == 0:
			npcSay(1632, "")
		case tk.verb == 4:
			npcSay(1627, "")
		case tk.subject > 0:
			npcAnswerOrScript(m+3, 1625)
		default:
			npcSay(1636, "")
		}
	case 6, 7:
		if q {
			npcSay(1631, "")
		} else {
			npcSay(1637, "")
		}
	case 8, 9:
		if !q {
			npcSay([2]int{1633, 1635}[tk.verb-8], "")
			return
		}
		switch {
		case ask[0] || tk.verb == 9 && ask[4]:
			npcSay(1630, "")
		case ask[1] && tk.verb == 8:
			npcAnswerOrScript(m+2, 1620)
		case ask[1]:
			npcSay(1634, "")
		case ask[3]:
			npcAnswerOrScript(m+4, 1622)
		case ask[4]:
			npcSay(1623, "")
		case ask[2]:
			npcSay(1621, "")
		default:
			npcAnswerOrScript(m+3, 1625)
		}
	}
}

// npcRespond (WMNPC d291) answers an interjection, then the question, remembering it.
func npcRespond() {
	if tk.interj != -1 && (tk.verb == -1 || tk.topic == -1) {
		npcAnswerInterjection()
	}
	if tk.verb != -1 && tk.topic != -1 {
		npcAnswer()
		tk.lastTopic, tk.lastPlural, tk.lastVerb, tk.lastAsk = tk.topic, tk.plural, tk.verb, -1
	}
}

// npcReadLine prompts member m (message 1513) in the text window and reads a line.
func npcReadLine(m int) []byte {
	prompt := replaceCharWithString(getMessage(1513), memberName(m), '^')
	printString(npcTextWin, prompt, 1)
	npcTextWin.CurX++
	s := npcInputLine(npcTextWin, 37-len(prompt), 6, 0)
	npcNewline(npcTextWin)
	return []byte(s)
}

// npc_input_word (WMNPC d53a) shows messages msg+2.. and reads a line from member m until
// it holds one of the choices of message msg ("Nword/Nword/...", N the script byte): that
// byte. once returns 1 for a line without a choice.
func npcInputWord(m, msg, once int) int {
	if dtLog != nil { // the difftest stubs it
		return 0
	}
	choices := getMessage(msg)
	for {
		npcSayAll(msg + 2)
		line := string(npcReplaceSynonyms(append(npcReadLine(m), ' ')))
		for i := 0; i < len(choices); {
			j := strings.IndexByte(choices[i+1:], '/')
			if j < 0 {
				j = len(choices) - i - 1
			}
			if w := choices[i+1 : i+1+j]; line != "" && strings.Contains(line, w) {
				return int(choices[i])
			}
			i += j + 2
		}
		if once == 1 {
			return 1
		}
	}
}

// npcTalk (WMNPC d674): member m talks with the NPC until an interjection or a script
// ends it.
func npcTalk(m int) {
	npcTalkMember = m
	tk = npcTalkState{lastTopic: -1, lastVerb: -1, lastAsk: -1}
	npcOpenText()
	setCursor(npcTextWin, npcScrollCol, 0)
	for !tk.done {
		line := npcReadLine(m)
		npcStripPunct(line)
		line = npcReplaceSynonyms(append(line, ' '))
		tk.topic, tk.plural, tk.verb, tk.interj = -1, 0, -1, -1
		tk.ask = [7]bool{}
		tk.negated, tk.question, tk.aboutSubject = false, tk.questionMark, false
		tk.subject, tk.object, tk.person = 0, 1, -1
		tk.topic2, tk.plural2, tk.verb2 = -1, 0, -1
		for i := 0; ; {
			for i < len(line) && line[i] == ' ' {
				i++
			}
			tk.prevClass = npcParseWord(line[i:])
			if i = npcWordEnd(line, i); i == -1 {
				break
			}
		}
		if len(line) > 0 && npcAnswerSpecial() == 0 {
			npcRespond()
		}
	}
	npcCloseText()
}
