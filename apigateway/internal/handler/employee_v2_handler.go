package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/locvowork/employee_management_sample/apigateway/internal/logger"
	"github.com/locvowork/employee_management_sample/apigateway/internal/service/serviceutils"
	"github.com/locvowork/employee_management_sample/apigateway/pkg/simpleexcelv2"
)

func (h *EmployeeHandler) ExportV2FromYAMLHandler(c echo.Context) error {
	// YAML configuration
	yamlConfig := ""
	data, err := os.ReadFile("report_config_v2.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to read YAML file", err)
	}
	yamlConfig = string(data)

	productSectionEditable := generateRandomProducts(500)
	productSectionOriginal := make([]Product, len(productSectionEditable))
	copy(productSectionOriginal, productSectionEditable)

	// Initialize exporter with inline config
	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to parse inline report config", err)
	}

	// Register a simple currency formatter for demonstration
	exporter.RegisterFormatter("currency", func(v interface{}) interface{} {
		if val, ok := v.(float64); ok {
			return fmt.Sprintf("$%.2f", val)
		}
		return v
	})

	// Bind data
	exporter.
		BindSectionData("product_section_editable", productSectionEditable).
		BindSectionData("product_section_original", productSectionOriginal)

	// Export to bytes
	excelBytes, err := exporter.ToBytes()
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to generate Excel file", err)
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="comparason_report.xlsx"`)
	c.Response().Header().Set("Content-Length", strconv.Itoa(len(excelBytes)))

	// Write response
	_, err = c.Response().Write(excelBytes)
	return err
}

func (h *EmployeeHandler) ExportLargeDataHandler(c echo.Context) error {
	// Generate large dataset
	count := 50000
	data := make([]Product, count)
	for i := 0; i < count; i++ {
		data[i] = Product{
			Name:      fmt.Sprintf("Product %d", i+1),
			Price:     10.0 + float64(i)*0.01,
			Category:  "Bulk Item",
			Available: i%2 == 0,
			Weight:    1.0,
			Color:     "Generic",
		}
	}

	// Create and configure exporter
	exporter := simpleexcelv2.NewExcelDataExporter().
		AddSheet("Large Export").
		AddSection(&simpleexcelv2.SectionConfig{
			Title:      fmt.Sprintf("Bulk Products Export (%d rows)", count),
			Data:       data,
			ShowHeader: true,
			Columns: []simpleexcelv2.ColumnConfig{
				{FieldName: "Name", Header: "Product Name", Width: 30},
				{FieldName: "Price", Header: "Unit Price", Width: 15},
				{FieldName: "Category", Header: "Category", Width: 20},
				{FieldName: "Available", Header: "In Stock", Width: 10},
			},
		}).
		Build()

	// Set headers for file download
	c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="large_products_%d.xlsx"`, count))

	// Stream directly to response
	return exporter.ToWriter(c.Response().Writer)
}

func generateRandomProducts(count int) []Product {
	categories := []string{"Electronics", "Home & Garden", "Sports", "Books", "Toys"}
	colors := []string{"Red", "Blue", "Black", "White", "Silver", "Gold"}

	products := make([]Product, count)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		products[i] = Product{
			Name:      fmt.Sprintf("Product %04d", i+1),
			Price:     10.0 + r.Float64()*990.0,
			Category:  categories[r.Intn(len(categories))],
			Available: r.Float64() > 0.3,
			Weight:    0.1 + r.Float64()*10.0,
			Color:     colors[r.Intn(len(colors))],
		}
	}
	return products
}

// LargeItem represents a struct with 36+ fields for performance testing
type LargeItem struct {
	ID              int
	Name            string
	Description     string
	LongDescription string // 3000+ characters
	Category        string
	SubCategory     string
	Status          string
	Priority        string
	Price           float64
	Cost            float64
	Margin          float64
	Quantity        int
	MinStock        int
	MaxStock        int
	Supplier        string
	Brand           string
	SKU             string
	Barcode         string
	Weight          float64
	Length          float64
	Width           float64
	Height          float64
	Color           string
	Material        string
	Origin          string
	Warehouse       string
	Location        string
	CreatedAt       string
	UpdatedAt       string
	CreatedBy       string
	UpdatedBy       string
	IsActive        bool
	IsFeatured      bool
	Rating          float64
	ReviewCount     int
	Tags            string
	Notes           string
	DetailedNotes   string // 3000+ characters
	TechnicalSpecs  string // 3000+ characters
}

func generateLargeText(r *rand.Rand, minChars int) string {
	words := []string{"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit", "sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore", "magna", "aliqua", "enim", "ad", "minim", "veniam", "quis", "nostrud", "exercitation", "ullamco", "laboris", "nisi", "aliquip", "ex", "ea", "commodo", "consequat", "duis", "aute", "irure", "in", "reprehenderit", "voluptate", "velit", "esse", "cillum", "fugiat", "nulla", "pariatur", "excepteur", "sint", "occaecat", "cupidatat", "non", "proident", "sunt", "culpa", "qui", "officia", "deserunt", "mollit", "anim", "id", "est", "laborum"}
	var sb strings.Builder
	for sb.Len() < minChars {
		sb.WriteString(words[r.Intn(len(words))])
		sb.WriteString(" ")
	}
	return sb.String()
}

func generateLargeItems(count int) []LargeItem {
	categories := []string{"Electronics", "Home & Garden", "Sports", "Books", "Toys", "Clothing", "Food", "Automotive"}
	statuses := []string{"Active", "Inactive", "Pending", "Archived"}
	priorities := []string{"High", "Medium", "Low"}
	colors := []string{"Red", "Blue", "Black", "White", "Silver", "Gold", "Green", "Yellow"}
	materials := []string{"Plastic", "Metal", "Wood", "Glass", "Fabric", "Leather"}
	warehouses := []string{"WH-A", "WH-B", "WH-C", "WH-D"}

	items := make([]LargeItem, count)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		price := 10.0 + r.Float64()*990.0
		cost := price * (0.4 + r.Float64()*0.3)
		items[i] = LargeItem{
			ID:              i + 1,
			Name:            fmt.Sprintf("Item %05d", i+1),
			Description:     fmt.Sprintf("Description for item %d with detailed information", i+1),
			LongDescription: generateLargeText(r, 3000+r.Intn(1000)),
			Category:        categories[r.Intn(len(categories))],
			SubCategory:     fmt.Sprintf("SubCat-%d", r.Intn(20)+1),
			Status:          statuses[r.Intn(len(statuses))],
			Priority:        priorities[r.Intn(len(priorities))],
			Price:           price,
			Cost:            cost,
			Margin:          price - cost,
			Quantity:        r.Intn(1000),
			MinStock:        r.Intn(50),
			MaxStock:        100 + r.Intn(900),
			Supplier:        fmt.Sprintf("Supplier-%d", r.Intn(50)+1),
			Brand:           fmt.Sprintf("Brand-%d", r.Intn(30)+1),
			SKU:             fmt.Sprintf("SKU-%08d", r.Intn(100000000)),
			Barcode:         fmt.Sprintf("%013d", r.Int63n(10000000000000)),
			Weight:          0.1 + r.Float64()*50.0,
			Length:          1.0 + r.Float64()*100.0,
			Width:           1.0 + r.Float64()*100.0,
			Height:          1.0 + r.Float64()*100.0,
			Color:           colors[r.Intn(len(colors))],
			Material:        materials[r.Intn(len(materials))],
			Origin:          fmt.Sprintf("Country-%d", r.Intn(50)+1),
			Warehouse:       warehouses[r.Intn(len(warehouses))],
			Location:        fmt.Sprintf("A%d-R%d-S%d", r.Intn(10)+1, r.Intn(50)+1, r.Intn(100)+1),
			CreatedAt:       time.Now().AddDate(0, 0, -r.Intn(365)).Format("2006-01-02 15:04:05"),
			UpdatedAt:       time.Now().AddDate(0, 0, -r.Intn(30)).Format("2006-01-02 15:04:05"),
			CreatedBy:       fmt.Sprintf("User%d", r.Intn(100)+1),
			UpdatedBy:       fmt.Sprintf("User%d", r.Intn(100)+1),
			IsActive:        r.Float64() > 0.2,
			IsFeatured:      r.Float64() > 0.8,
			Rating:          1.0 + r.Float64()*4.0,
			ReviewCount:     r.Intn(500),
			Tags:            fmt.Sprintf("tag1,tag%d,tag%d", r.Intn(50)+2, r.Intn(100)+51),
			Notes:           fmt.Sprintf("Notes for item %d", i+1),
			DetailedNotes:   generateLargeText(r, 3000+r.Intn(1000)),
			TechnicalSpecs:  generateLargeText(r, 3000+r.Intn(1000)),
		}
	}
	return items
}

// ExportLargeColumnHandler exports data with 36 columns and thousands of rows for performance testing
func (h *EmployeeHandler) ExportLargeColumnHandler(c echo.Context) error {
	start := time.Now()
	ctx := c.Request().Context()

	// Read YAML configuration
	data, err := os.ReadFile("report_config_perf.yaml")
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to read YAML file", err)
	}
	yamlConfig := string(data)

	// Generate 2000 items with 36+ columns each (reduced from 5000 to keep multi-section export snappy)
	count := 2000
	editableItems := generateLargeItems(count)

	// Initialize exporter
	exporter, err := simpleexcelv2.NewExcelDataExporterFromYamlConfig(yamlConfig)
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to parse YAML config", err)
	}

	// Start Stream
	streamer, err := exporter.StartStream(c.Response())
	if err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to start stream", err)
	}

	// Set headers for file download and keep-alive before starting heavy write
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="large_column_multi_section_perf_%d_rows.xlsx"`, count))
	// Content-Length is unknown for streaming

	// Write Editable Data in batches
	batchSize := 500
	for i := 0; i < len(editableItems); i += batchSize {
		end := i + batchSize
		if end > len(editableItems) {
			end = len(editableItems)
		}
		if err := streamer.Write("large_column_editable", editableItems[i:end]); err != nil {
			return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to write editable batch", err)
		}
	}

	// Close Stream
	if err := streamer.Close(); err != nil {
		return serviceutils.ResponseError(c, http.StatusInternalServerError, "Failed to close stream", err)
	}

	elapsed := time.Since(start)
	logger.DebugLog(ctx, "ExportLargeColumnHandler elapsed: %s", elapsed.String())

	return nil
}
