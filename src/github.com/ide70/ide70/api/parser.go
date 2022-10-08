package api

import (
	"fmt"
	"regexp"
	"strings"
)

type Parser struct {
	text      string
	splitter  Splitter
	extractor *RegexpExtractor
}

type Splitter interface {
	tokens(text string) (stream chan string)
}

type SplitBySep struct {
	sep string
}

type RegexpExtractor struct {
	scanPattern string
	keyList     []string
	typeConv    []string
}

func (bd *BinaryData) CreateParser() *Parser {
	return &Parser{text: string(*bd.GetData())}
}

func (p *Parser) SetSplitBySep(sep string) *Parser {
	p.splitter = &SplitBySep{sep: sep}
	return p
}

func (sbsp *SplitBySep) tokens(text string) (stream chan string) {
	stream = make(chan string)

	go func() {
		for _, t := range strings.Split(text, sbsp.sep) {
			stream <- t
		}
		close(stream)
	}()
	return
}

func (p *Parser) SetExtractor(scanPattern, keyList, typeConv string) *Parser {
	p.extractor = &RegexpExtractor{scanPattern: scanPattern, keyList: strings.Split(keyList, ","), typeConv: strings.Split(typeConv, ",")}
	return p
}

func (p *Parser) Extract() ITable {
	return p.extractor.extractTokens(p.text)
}


func convertByFormat(s, format string) interface{} {
	switch format {
		case "%s":
		return s
		case "%d":
		var d int
		fmt.Sscanf(s, format, &d)
		return d
		case "%b":
		var d int
		fmt.Sscanf(s, format, &d)
		return d == 1
	}
	return nil
}

// VALUES \('([^']*)', '([^']*)', '([^']*)', '([^']*)', ([0-9]+)\)

func (extr *RegexpExtractor) extractTokens(text string) ITable {
	re, err := regexp.Compile(extr.scanPattern)
	if err != nil {
		logger.Error(err)
		return nil
	}
	logger.Debug("extractTokens, search:", extr.scanPattern)
	hits := re.FindAllStringSubmatch(text, -1)
	logger.Debug("extractTokens, nrhits:", len(hits))
	t := ITable{}
	for _, hit := range hits {
		simap := SIMap{}
		for i := 1; i < len(hit); i++ {
			convVal := convertByFormat(hit[i], extr.typeConv[i-1])
			logger.Debug("extractTokens, cvt res:", convVal)
			simap[extr.keyList[i-1]] = convVal
		}
		t = append(t, simap)
	}
	logger.Debug("extractTokens, result:", t)
	return t
}

// "VALUES ('%s', '%s', '%s', '%d', %d)"
// VALUES ('Betekintő', 'PTLKER09', 'Adatlap.xml', 'L', 1)
