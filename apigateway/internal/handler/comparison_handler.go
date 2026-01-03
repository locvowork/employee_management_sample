package handler

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/dataflow"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/pipeline"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcelv2"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcelv3"
)

type WikiPerson struct {
	Name string `json:"name" excel:"Name"`
	URL  string `json:"url" excel:"URL"`
}

type ComparisonHandler struct{}

func NewComparisonHandler() *ComparisonHandler {
	return &ComparisonHandler{}
}

// Global regex to match potential names in Wikipedia list pages
// This is a simplified regex for demo purposes.
var nameRegex = regexp.MustCompile(`<li><a href="(/wiki/[^"]+)" title="([^"]+)">([^<]+)</a>`)

// --- Helper Functions ---

func fetchWikiPage(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AntigravityScraper/1.0; +http://localhost:8082)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func parseWikiNames(body string) []WikiPerson {
	matches := nameRegex.FindAllStringSubmatch(body, -1)
	var people []WikiPerson
	for _, match := range matches {
		if len(match) >= 4 {
			people = append(people, WikiPerson{
				Name: match[3],
				URL:  "https://en.wikipedia.org" + match[1],
			})
		}
	}
	return people
}

// --- TPL Style Implementation (pkg/pipeline) ---

func (h *ComparisonHandler) ExportWikiTPL(c echo.Context) error {
	wikiURLs := []string{
		"https://en.wikipedia.org/wiki/List_of_computer_scientists",
		"https://en.wikipedia.org/wiki/List_of_American_mathematicians",
		"https://en.wikipedia.org/wiki/Timeline_of_ancient_Greek_mathematicians",
	}
	ctx := c.Request().Context()
	logger.InfoLog(ctx, "Exporting wiki names (TPL Style)")
	start := time.Now()
	// 1. Create Blocks
	buffer := pipeline.NewBufferBlock(pipeline.WithBufferSize(10))

	fetchingRetry := pipeline.NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			url := input.(string)
			logger.InfoLog(ctx, "Fetching URL: %s", url)
			return fetchWikiPage(url)
		},
		pipeline.WithRetryPolicy(pipeline.RetryPolicy{
			MaxRetries: 3,
			Backoff:    100 * time.Millisecond,
		}),
	)

	parser := pipeline.NewTransformBlock(func(input interface{}) (interface{}, error) {
		body := input.(string)
		logger.InfoLog(ctx, "Parsing body...")
		return parseWikiNames(body), nil
	})

	var allPeople []WikiPerson
	collector := pipeline.NewActionBlock(func(input interface{}) error {
		people := input.([]WikiPerson)
		logger.InfoLog(ctx, "Collecting people: %#v", people)
		allPeople = append(allPeople, people...)
		return nil
	})

	// 2. Link
	pipeline.LinkTo(buffer, fetchingRetry, nil)
	pipeline.LinkTo(fetchingRetry, parser, nil)
	pipeline.LinkTo(parser, collector, nil)

	// 3. Execution
	go func() {
		for _, url := range wikiURLs {
			logger.InfoLog(ctx, "Posting URL: %s", url)
			buffer.Post(url)
		}
		buffer.Complete()
	}()
	logger.InfoLog(ctx, "Pipeline started")
	// 4. Wait
	err := pipeline.WaitAll(buffer, fetchingRetry, parser, collector)

	if err != nil {
		logger.ErrorLog(ctx, "Pipeline failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logger.InfoLog(ctx, "TPL Pipeline finished in %v, collected %d people", time.Since(start), len(allPeople))

	if len(allPeople) == 0 {
		logger.WarnLog(ctx, "No people collected from Wikipedia")
	}

	return h.exportToExcel(c, allPeople, "wiki_names_tpl.xlsx")
}

// --- Idiomatic Go Style Implementation (pkg/dataflow) ---

func (h *ComparisonHandler) ExportWikiIdiomatic(c echo.Context) error {
	ctx := c.Request().Context()
	logger.InfoLog(ctx, "Exporting wiki names (Idiomatic Style)")
	start := time.Now()
	wikiURLs := []interface{}{
		"https://en.wikipedia.org/wiki/List_of_computer_scientists",
		"https://en.wikipedia.org/wiki/List_of_American_mathematicians",
		"https://en.wikipedia.org/wiki/Timeline_of_ancient_Greek_mathematicians",
	}

	// 1. Source
	src := dataflow.From(ctx, wikiURLs...)

	// 2. Fetch (Parallel) with Retry
	bodies := dataflow.Map(ctx, src, func(msg interface{}) (interface{}, error) {
		return fetchWikiPage(msg.(string))
	}, dataflow.WithWorkers(2), dataflow.WithRetry(3, dataflow.ExponentialBackoff(100*time.Millisecond)))

	// 3. Parse
	parsed := dataflow.Map(ctx, bodies, func(msg interface{}) (interface{}, error) {
		return parseWikiNames(msg.(string)), nil
	})

	// 4. Collect
	var allPeople []WikiPerson
	err := dataflow.ForEach(ctx, parsed, func(msg interface{}) error {
		people := msg.([]WikiPerson)
		allPeople = append(allPeople, people...)
		return nil
	})

	if err != nil {
		logger.ErrorLog(ctx, "Idiomatic Pipeline failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	logger.InfoLog(ctx, "Idiomatic Pipeline finished in %v, collected %d people", time.Since(start), len(allPeople))

	return h.exportToExcel(c, allPeople, "wiki_names_idiomatic.xlsx")
}

func (h *ComparisonHandler) ExportWikiStreaming(c echo.Context) error {
	ctx := c.Request().Context()
	logger.InfoLog(ctx, "Exporting wiki names (Streaming Style)")
	start := time.Now()
	wikiURLs := []interface{}{
		"https://en.wikipedia.org/wiki/List_of_computer_scientists",
		"https://en.wikipedia.org/wiki/List_of_American_mathematicians",
		"https://en.wikipedia.org/wiki/Timeline_of_ancient_Greek_mathematicians",
	}

	// 1. Prepare Exporter
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=wiki_names_streaming.xlsx")
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	exporter := simpleexcelv3.NewStreamExporter(c.Response().Writer)
	sheet, err := exporter.AddSheet("Wikipedia People")
	if err != nil {
		return err
	}

	cols := []simpleexcelv3.ColumnConfig{
		{FieldName: "Name", Header: "Person Name", Width: 40},
		{FieldName: "URL", Header: "Wiki URL", Width: 60},
	}
	if err := sheet.WriteHeader(cols); err != nil {
		return err
	}

	// 2. Dataflow Pipeline
	src := dataflow.From(ctx, wikiURLs...)

	bodies := dataflow.Map(ctx, src, func(msg interface{}) (interface{}, error) {
		return fetchWikiPage(msg.(string))
	}, dataflow.WithWorkers(2), dataflow.WithRetry(3, dataflow.ExponentialBackoff(100*time.Millisecond)))

	parsed := dataflow.Map(ctx, bodies, func(msg interface{}) (interface{}, error) {
		return parseWikiNames(msg.(string)), nil
	})

	// 3. ForEach + AppendBatch (appending each parsed slice immediately)
	var count int
	err = dataflow.ForEach(ctx, parsed, func(msg interface{}) error {
		people := msg.([]WikiPerson)
		count += len(people)
		logger.InfoLog(ctx, "Appending batch of %d people to Excel", len(people))
		return sheet.WriteBatch(people)
	})

	if err != nil {
		logger.ErrorLog(ctx, "Streaming Pipeline failed: %v", err)
		// Note: At this point headers might already be sent, so we can't easily return JSON error
		return nil
	}

	if err := exporter.Close(); err != nil {
		logger.ErrorLog(ctx, "Failed to close exporter: %v", err)
		return nil
	}

	logger.InfoLog(ctx, "Streaming Pipeline finished in %v, exported %d people", time.Since(start), count)
	return nil
}

func (h *ComparisonHandler) exportToExcel(c echo.Context, data []WikiPerson, filename string) error {
	exporter := simpleexcelv2.NewExcelDataExporter()

	sheet := exporter.AddSheet("Wikipedia People")

	section := &simpleexcelv2.SectionConfig{
		Title: "Extracted Names from Wikipedia",
		Columns: []simpleexcelv2.ColumnConfig{
			{FieldName: "Name", Header: "Person Name", Width: 40},
			{FieldName: "URL", Header: "Wiki URL", Width: 60},
		},
		Data: data,
	}

	sheet.AddSection(section)

	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", filename))
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	return exporter.ToWriter(c.Response().Writer)
}
