package gown

import (
	"fmt"
	"os"
	"strings"
)

type WN struct {
	senseIndex  senseIndex
	posIndicies map[int]dataIndex
	posData     map[int]dataFile
	exceptions  []map[string]string
}

func GetWordNetDictDir() (string, error) {
	systemDefaults := []string{
		"/usr/WordNet-3.%d/dict",
		"/usr/share/WordNet-3.%d/dict",
		"/usr/local/WordNet-3.%d/dict",
		"/usr/local/share/WordNet-3.%d/dict",
		"/opt/WordNet-3.%d/dict",
		"/opt/share/WordNet-3.%d/dict",
		"/opt/local/WordNet-3.%d/dict",
		"/opt/local/share/WordNet-3.%d/dict",
	}
	// check environment variables
	dictname := os.Getenv("WNHOME") + "/dict"
	_, err := os.Stat(dictname)
	if err == nil {
		return dictname, nil
	}

	dictname = os.Getenv("WNSEARCHDIR")
	_, err = os.Stat(dictname)
	if err == nil {
		return dictname, nil
	}

	// check possible installation dirs
	for v := 0; v <= 1; v++ { // checks for WordNet 3.0 and 3.1
		for _, systemDefault := range systemDefaults {
			dictname = fmt.Sprintf(systemDefault, v)
			_, err = os.Stat(dictname)
			if err == nil {
				return dictname, nil
			}
		}
	}

	// tried everything
	return "", fmt.Errorf("Can't find WordNet dictionary")
}

func LoadWordNet(dictDirname string) (*WN, error) {
	wn := &WN{
		senseIndex:  nil,
		posIndicies: map[int]dataIndex{},
		posData:     map[int]dataFile{},
	}

	var err error = nil
	pos_file_names := []string{"", "noun", "verb", "adj", "adv"}
	for i := 1; i < len(pos_file_names); i++ {
		wn.posIndicies[i], err = readPosIndex(dictDirname + "/index." + pos_file_names[i])
		if err != nil {
			return nil, err
		}
		wn.posData[i], err = readPosData(dictDirname + "/data." + pos_file_names[i])
		if err != nil {
			return nil, err
		}
	}

	wn.senseIndex, err = loadSenseIndex(wn, dictDirname + "/index.sense")
	if err != nil {
		return nil, err
	}

	return wn, nil
}

func (wn *WN) LookupWithPartOfSpeech(lemma string, pos int) *DataIndexEntry {
	posIndexPtr, exists := wn.posIndicies[pos]
	if !exists {
		return nil
	}
	sn, exists := posIndexPtr[strings.ToLower(lemma)]
	if exists {
		return &sn
	} else {
		return nil
	}
}

func (wn *WN) LookupSensesWithPartOfSpeech(lemma string, pos int) []*SenseIndexEntry {
	senses, _ := wn.senseIndex[lemma]
	ret := make([]*SenseIndexEntry, 0, len(senses))
	for i, _ := range senses {
		if senses[i].partOfSpeech == pos {
			ret = append(ret, senses[i])
		}
	}
	return ret
}

func (wn *WN) LookupWithPartOfSpeechAndSense(lemma string, pos int, senseId int) *SenseIndexEntry {
	senses, _ := wn.senseIndex[lemma]
	for _, sense := range senses {
		if (sense.partOfSpeech == pos) && (sense.senseNumber == senseId) {
			return sense
		}
	}
	return nil
}

func (wn *WN) Lookup(lemma string) []*SenseIndexEntry {
	senseEntries, _ := wn.senseIndex[strings.ToLower(lemma)]
	return senseEntries
}

func (wn *WN) GetSynset(pos int, synsetOffset int) *Synset {
	if pos == POS_ADJECTIVE_SATELLITE {
		pos = POS_ADJECTIVE
	}
	idxPtr, exists := wn.posData[pos]
	if !exists || idxPtr == nil {
		return nil
	}
	s, _ := idxPtr[synsetOffset]
	return s
}

func (wn *WN) Iter() <-chan *Synset {
	outChan := make(chan *Synset)
	go func() {
		for _, datFile := range wn.posData {
			for _, synset := range datFile {
				outChan <- synset
			}
		}
		close(outChan)
	}()
	return outChan
}

func (wn *WN) IterSenses() <-chan *SenseIndexEntry {
	outchan := make(chan *SenseIndexEntry)
	go func() {
		for _, senses := range wn.senseIndex {
			for _, sense := range senses {
				outchan <- sense
			}
		}
		close(outchan)
	}()
	return outchan
}

type DataIndexPair struct {
    Lexeme string
    IndexEntry *DataIndexEntry
}
func (wn *WN) IterDataIndex(pos int) <-chan DataIndexPair {
	table, ok := wn.posIndicies[pos]
	if !ok {
		return nil
	}
	out := make(chan DataIndexPair)
	go func() {
		for k, v := range table {
			out <- DataIndexPair {
				Lexeme: k,
				IndexEntry: &v,
			}
		}
		close(out)
	}()
	return out
}
