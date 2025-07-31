package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type Processor struct {
	categoryCount map[string]int
}

type MailHeader struct {
	From      string   `json:"From"`
	MessageID string   `json:"Message-ID"`
	Received  []string `json:"Received"`
	Subject   string   `json:"Subject"`
	To        []string `json:"To"`
}

type MailInfo struct {
	MailHeader  MailHeader `json:"MailHeader"`
	Attachments int        `json:"Attachments"`
}

type Content struct {
	Id               int     `json:"Id"`
	ContentType      string  `json:"ContentType"`
	Name             string  `json:"Name"`
	OriginalSize     int     `json:"OriginalSize"`
	UnpackedSize     int     `json:"UnpackedSize"`
	CompressionRatio float64 `json:"CompressionRatio"`
	ContentMode      string  `json:"ContentMode,omitempty"`
	NotScanned       string  `json:"NotScanned,omitempty"`
}

type MetaData struct {
	MailInfo       *MailInfo `json:"MailInfo,omitempty"`
	Content        []Content `json:"Content,omitempty"`
	FilesContained int       `json:"FilesContained,omitempty"`
}

type Indicator struct {
	Item        string  `json:"Item"`
	Info        string  `json:"Info,omitempty"`
	Description string  `json:"Description"`
	Rating      float64 `json:"Rating"`
}

type IndicatorCategory struct {
	Category   string      `json:"Category"`
	Indicators []Indicator `json:"Indicators"`
}

type Object struct {
	Name                string              `json:"Name"`
	Id                  int                 `json:"Id"`
	ParentId            int                 `json:"ParentId"`
	Sha256              string              `json:"Sha256,omitempty"`
	ObjectType          string              `json:"ObjectType"`
	ObjectSize          int                 `json:"ObjectSize"`
	Rating              float64             `json:"Rating"`
	MetaData            *MetaData           `json:"MetaData,omitempty"`
	IndicatorCategories []IndicatorCategory `json:"IndicatorCategories,omitempty"`
	Objects             []Object            `json:"Objects,omitempty"`
}

type SummaryDescription struct {
	Description string `json:"Description"`
	RatingFlag  string `json:"RatingFlag"`
}

type RootObject struct {
	Name                string              `json:"Name"`
	Id                  int                 `json:"Id"`
	ParentId            int                 `json:"ParentId"`
	Sha256              string              `json:"Sha256"`
	ObjectType          string              `json:"ObjectType"`
	ObjectSize          int                 `json:"ObjectSize"`
	Rating              float64             `json:"Rating"`
	MetaData            MetaData            `json:"MetaData"`
	IndicatorCategories []IndicatorCategory `json:"IndicatorCategories"`
	Objects             []Object            `json:"Objects"`
}

type ScanResult struct {
	FileName           string             `json:"FileName"`
	Sha256             string             `json:"Sha256"`
	ScanResult         string             `json:"ScanResult"`
	ScanTime           int                `json:"ScanTime"`
	TimeStamp          string             `json:"TimeStamp"`
	SummaryDescription SummaryDescription `json:"SummaryDescription"`
	RootObject         RootObject         `json:"RootObject"`
}

func NewProcessor() *Processor {
	return &Processor{
		categoryCount: make(map[string]int),
	}
}

func (p *Processor) countCategories(categories []IndicatorCategory) {
	for _, cat := range categories {
		p.categoryCount[cat.Category]++
	}
}

func (p *Processor) collectAndCountCategories(obj Object) {
	p.countCategories(obj.IndicatorCategories)
	for _, sub := range obj.Objects {
		p.collectAndCountCategories(sub)
	}
}

func printIndicators(indicatorCategories []IndicatorCategory) {
	for _, cat := range indicatorCategories {
		fmt.Fprintln(os.Stdout, "  - Category:", cat.Category)
		for _, ind := range cat.Indicators {
			fmt.Fprintln(os.Stdout, "  - Indicator:", ind.Description, "("+ind.Item+")")
		}
	}
}

func printObject(obj Object) {
	fmt.Fprintf(os.Stdout, "Object: %s (Type: %s, Rating: %.2f)\n", obj.Name, obj.ObjectType, obj.Rating)
	printIndicators(obj.IndicatorCategories)
}

func printRootObject(root RootObject) {
	fmt.Fprintf(os.Stdout, "RootObject: %s (Type: %s, Rating: %.2f)\n", root.Name, root.ObjectType, root.Rating)
	printIndicators(root.IndicatorCategories)
}

func traverseObjectsVerbose(objs []Object, minRating float64) {
	for _, obj := range objs {
		if obj.Rating >= minRating {
			printObject(obj)
		}
		if len(obj.Objects) > 0 {
			traverseObjectsVerbose(obj.Objects, minRating)
		}
	}
}

func findHighestRating(objs []Object, current float64) float64 {
	max := current
	for _, obj := range objs {
		if obj.Rating > max {
			max = obj.Rating
		}
		if len(obj.Objects) > 0 {
			subMax := findHighestRating(obj.Objects, max)
			if subMax > max {
				max = subMax
			}
		}
	}
	return max
}

func collectDetections(root RootObject) []string {
	var detections []string
	all := append([]IndicatorCategory{}, root.IndicatorCategories...)
	for _, obj := range root.Objects {
		all = append(all, collectObjectCategories(obj)...)
	}
	for _, cat := range all {
		if cat.Category == "Detected" {
			for _, indi := range cat.Indicators {
				detections = append(detections, fmt.Sprintf("Type=%s name=%s", indi.Item, indi.Info))
			}
			break
		}
	}
	if len(detections) == 0 {
		detections = append(detections, "none")
	}
	return detections
}

func collectObjectCategories(obj Object) []IndicatorCategory {
	cats := obj.IndicatorCategories
	for _, sub := range obj.Objects {
		cats = append(cats, collectObjectCategories(sub)...)
	}
	return cats
}

func (p *Processor) processEntry(msgID string, jsonData string, minRating float64, verbose bool) {
	var result ScanResult
	err := json.Unmarshal([]byte(jsonData), &result)
	if err != nil {
		fmt.Fprintln(os.Stdout, "Varist-RC=1 Rating=UNKNOWN Result=ParseError Detections=")
		return
	}

	p.countCategories(result.RootObject.IndicatorCategories)
	for _, obj := range result.RootObject.Objects {
		p.collectAndCountCategories(obj)
	}

	if verbose {
		fmt.Fprintf(os.Stdout, "=== msgid: %s ===\n", msgID)
		fmt.Fprintln(os.Stdout, "File:", result.FileName)
		fmt.Fprintln(os.Stdout, "Scan result:", result.ScanResult)
		fmt.Fprintf(os.Stdout, "Scan time: %d ms (%.2f s)\n", result.ScanTime, float64(result.ScanTime)/1000.0)
		fmt.Fprintln(os.Stdout, "Description:", result.SummaryDescription.Description)
		fmt.Fprintln(os.Stdout, "RatingFlag:", result.SummaryDescription.RatingFlag)
		fmt.Fprintf(os.Stdout, "RootObject rating: %.2f\n", result.RootObject.Rating)

		if result.RootObject.Rating >= minRating {
			printRootObject(result.RootObject)
		}
		traverseObjectsVerbose(result.RootObject.Objects, minRating)
	} else {
		highest := findHighestRating(result.RootObject.Objects, result.RootObject.Rating)
		var finalResult string
		switch {
		case highest < 0.0001:
			finalResult = "HA-CLEAN"
		case highest >= 99.9:
			finalResult = "HA-VIRUS"
		default:
			finalResult = fmt.Sprintf("%.0f", highest)
		}

		detections := collectDetections(result.RootObject)
		fmt.Fprintf(os.Stdout, "Varist-RC=0 Rating=\"%s\" Result=%s Detections=%s\n",
			result.SummaryDescription.RatingFlag,
			finalResult,
			strings.Join(detections, ","),
		)
	}
}

func (p *Processor) ProcessFile(reader io.Reader, minRating float64, verbose bool) error {
	scanner := bufio.NewScanner(reader)
	var msgID string
	var jsonBuilder strings.Builder
	parsing := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "msgid=") {
			if parsing {
				p.processEntry(msgID, jsonBuilder.String(), minRating, verbose)
				jsonBuilder.Reset()
			}
			msgID = strings.TrimPrefix(line, "msgid=")
			parsing = true
		} else if parsing {
			jsonBuilder.WriteString(line)
		}
	}

	if parsing && jsonBuilder.Len() > 0 {
		p.processEntry(msgID, jsonBuilder.String(), minRating, verbose)
	}

	return scanner.Err()
}

func (p *Processor) ProcessResponse(reader io.Reader, msgID string, minRating float64, verbose bool) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	p.processEntry(msgID, string(data), minRating, verbose)
	return nil
}

func (p *Processor) PrintCategoryCount() {
	fmt.Fprintln(os.Stdout, "Category counts:")
	for k, v := range p.categoryCount {
		fmt.Fprintf(os.Stdout, "  %s: %d\n", k, v)
	}
}
