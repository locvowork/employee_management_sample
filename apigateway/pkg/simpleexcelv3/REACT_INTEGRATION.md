# React v17 Integration Guide

This guide demonstrates how to download large Excel files from the `simpleexcelv2` Go backend using React v17. It focuses on memory efficiency and handling potential timeouts.

## Core Concepts

1.  **Blob Handling**: Large files should be handled as Blobs to prevent browser memory issues.
2.  **ObjectURL Cleanup**: Always revoke created object URLs to prevent memory leaks.
3.  **Timeouts**: Use `AbortController` (Fetch API) or specialized timeout settings (Axios).
4.  **Download Tracking**: For very large files, provide visual feedback (loading state).
5.  **Backend Integration**: Use the correct endpoint paths and method names from `simpleexcelv2`.

## Backend Setup

First, ensure your Go backend is properly configured:

```go
// In your Go handler
func exportHandler(c echo.Context) error {
    exporter := simpleexcelv2.NewExcelDataExporter()

    // Configure your exporter
    exporter.AddSheet("Employees").
        AddSection(&simpleexcelv2.SectionConfig{
            Title:      "Team Members",
            Data:       employees,
            ShowHeader: true,
            Columns: []simpleexcelv2.ColumnConfig{
                {FieldName: "ID", Header: "Employee ID", Width: 15},
                {FieldName: "Name", Header: "Full Name", Width: 25},
                {FieldName: "Role", Header: "Position", Width: 20},
            },
        })

    // Set headers for file download
    c.Response().Header().Set(echo.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="employees.xlsx"`)

    // Stream directly to response
    return exporter.ToWriter(c.Response().Writer)
}
```

## Implementation with Axios (Recommended)

Axios is recommended for large downloads because of its built-in `onDownloadProgress` and easy timeout configuration.

```javascript
import React, { useState } from "react";
import axios from "axios";

const ExcelDownloadButton = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const handleDownload = async () => {
    setLoading(true);
    setError(null);

    // Create an AbortController for manual cancellation or timeout
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minute timeout

    try {
      const response = await axios({
        url: "http://localhost:8080/api/v2/employees/export/large", // Your endpoint
        method: "GET",
        responseType: "blob", // IMPORTANT: Handle response as a Blob
        signal: controller.signal,
        timeout: 300000, // External timeout (5 minutes)
        onDownloadProgress: (progressEvent) => {
          const percentage = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          );
          console.log(`Download progress: ${percentage}%`);
        },
      });

      clearTimeout(timeoutId);

      // Create a link element to trigger the download
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement("a");
      link.href = url;

      // Extract filename from Content-Disposition if available, or use default
      const contentDisposition = response.headers["content-disposition"];
      let filename = "export.xlsx";
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="(.+)"/);
        if (filenameMatch && filenameMatch.length > 1) {
          filename = filenameMatch[1];
        }
      }

      link.setAttribute("download", filename);
      document.body.appendChild(link);
      link.click();

      // CLEANUP: Important for memory management
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      if (err.name === "AbortError" || axios.isCancel(err)) {
        setError("Download timed out or was cancelled.");
      } else {
        setError("Failed to download Excel file. Please try again.");
        console.error(err);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <button
        onClick={handleDownload}
        disabled={loading}
        className="download-btn"
      >
        {loading ? "Generating Report..." : "Download Large Excel"}
      </button>
      {error && <p style={{ color: "red" }}>{error}</p>}
    </div>
  );
};

export default ExcelDownloadButton;
```

## Implementation with Fetch API

If you prefer not to use Axios, you can use the native `Fetch` API with `AbortController`.

```javascript
const downloadWithFetch = async () => {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 300000); // 5 minutes

  try {
    const response = await fetch("/api/v2/employees/export/large", {
      method: "GET",
      signal: controller.signal,
    });

    if (!response.ok) throw new Error("Network response was not ok");

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);

    const a = document.createElement("a");
    a.href = url;
    a.download = "export.xlsx";
    document.body.appendChild(a);
    a.click();

    // Memory safety: cleanup
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  } catch (err) {
    console.error("Fetch download failed", err);
  } finally {
    clearTimeout(timeoutId);
  }
};
```

## Advanced React Component with Progress

```javascript
import React, { useState, useRef } from "react";
import axios from "axios";

const ExcelDownloadWithProgress = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState(0);
  const [filename, setFilename] = useState("");
  const controllerRef = useRef(null);

  const handleDownload = async () => {
    setLoading(true);
    setError(null);
    setProgress(0);
    setFilename("");

    controllerRef.current = new AbortController();
    const timeoutId = setTimeout(() => controllerRef.current.abort(), 600000); // 10 minute timeout

    try {
      const response = await axios({
        url: "/api/v2/employees/export/large",
        method: "GET",
        responseType: "blob",
        signal: controllerRef.current.signal,
        timeout: 600000,
        onDownloadProgress: (progressEvent) => {
          const percentage = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          );
          setProgress(percentage);
        },
      });

      clearTimeout(timeoutId);

      // Extract filename
      const contentDisposition = response.headers["content-disposition"];
      let extractedFilename = "export.xlsx";
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="(.+)"/);
        if (filenameMatch && filenameMatch.length > 1) {
          extractedFilename = filenameMatch[1];
        }
      }
      setFilename(extractedFilename);

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", extractedFilename);
      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      if (err.name === "AbortError" || axios.isCancel(err)) {
        setError("Download timed out or was cancelled.");
      } else {
        setError("Failed to download Excel file. Please try again.");
        console.error(err);
      }
    } finally {
      setLoading(false);
      setProgress(0);
    }
  };

  const handleCancel = () => {
    if (controllerRef.current) {
      controllerRef.current.abort();
    }
  };

  return (
    <div className="download-container">
      <button
        onClick={handleDownload}
        disabled={loading}
        className="download-btn"
      >
        {loading ? "Generating Report..." : "Download Large Excel"}
      </button>

      {loading && (
        <div className="progress-container">
          <div className="progress-bar">
            <div
              className="progress-fill"
              style={{ width: `${progress}%` }}
            ></div>
          </div>
          <span className="progress-text">{progress}%</span>
          <button onClick={handleCancel} className="cancel-btn">
            Cancel
          </button>
        </div>
      )}

      {filename && (
        <p className="success-message">Download completed: {filename}</p>
      )}

      {error && <p className="error-message">{error}</p>}
    </div>
  );
};

export default ExcelDownloadWithProgress;
```

## TypeScript Integration

```typescript
import React, { useState } from "react";
import axios, { AxiosProgressEvent } from "axios";

interface DownloadProps {
  endpoint: string;
  filename?: string;
  timeout?: number;
}

const ExcelDownloadButton: React.FC<DownloadProps> = ({
  endpoint,
  filename = "export.xlsx",
  timeout = 300000,
}) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);

  const handleDownload = async () => {
    setLoading(true);
    setError(null);
    setProgress(0);

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await axios({
        url: endpoint,
        method: "GET",
        responseType: "blob",
        signal: controller.signal,
        timeout,
        onDownloadProgress: (progressEvent: AxiosProgressEvent) => {
          if (progressEvent.total) {
            const percentage = Math.round(
              (progressEvent.loaded * 100) / progressEvent.total
            );
            setProgress(percentage);
          }
        },
      });

      clearTimeout(timeoutId);

      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement("a");
      link.href = url;

      // Extract filename from response headers
      const contentDisposition = response.headers["content-disposition"];
      let downloadFilename = filename;
      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="(.+)"/);
        if (filenameMatch && filenameMatch.length > 1) {
          downloadFilename = filenameMatch[1];
        }
      }

      link.setAttribute("download", downloadFilename);
      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      if (axios.isCancel(err)) {
        setError("Download was cancelled.");
      } else if (err instanceof Error) {
        setError(`Download failed: ${err.message}`);
      } else {
        setError("An unknown error occurred.");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <button
        onClick={handleDownload}
        disabled={loading}
        className="download-btn"
      >
        {loading ? `Generating Report... ${progress}%` : "Download Excel"}
      </button>
      {error && <p style={{ color: "red" }}>{error}</p>}
    </div>
  );
};

export default ExcelDownloadButton;
```

## Performance & Memory Checklist

- [ ] **Response Type**: Always set `responseType: 'blob'` (Axios) or use `response.blob()` (Fetch).
- [ ] **Revoke ObjectURL**: `window.URL.revokeObjectURL(url)` is critical. Without it, the browser keeps the file data in memory until the tab is closed.
- [ ] **Timeout**: Large exports can take minutes. Ensure your frontend timeout is longer than your backend processing time.
- [ ] **Chunking**: For truly massive data (millions of rows), consider using the **CSV export** mode (`ToCSV`) to reduce the browser's parsing overhead.
- [ ] **Background Processing**: If a download takes more than 1 minute, consider moving to an "Export Job" pattern where the backend generates the file, stores it, and notifies the user via WebSocket or polling.

## Error Handling Best Practices

```javascript
const handleExportError = (error) => {
  if (error.name === "AbortError") {
    return "Download was cancelled or timed out.";
  }

  if (error.response) {
    // Server responded with error status
    const status = error.response.status;
    switch (status) {
      case 404:
        return "Export endpoint not found.";
      case 413:
        return "Request too large. Please try with a smaller dataset.";
      case 500:
        return "Server error occurred. Please try again later.";
      default:
        return `Server error (${status}). Please try again.`;
    }
  }

  if (error.request) {
    // Network error
    return "Network error. Please check your connection.";
  }

  return "An unexpected error occurred.";
};
```

## Integration with State Management

```javascript
// Using Redux Toolkit
import { createSlice, createAsyncThunk } from "@reduxjs/toolkit";

export const downloadExcel = createAsyncThunk(
  "export/downloadExcel",
  async ({ endpoint, filename }, { rejectWithValue }) => {
    try {
      const response = await axios({
        url: endpoint,
        method: "GET",
        responseType: "blob",
        timeout: 600000,
      });

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", filename);
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      return { success: true };
    } catch (error) {
      return rejectWithValue(handleExportError(error));
    }
  }
);

const exportSlice = createSlice({
  name: "export",
  initialState: {
    loading: false,
    error: null,
    success: false,
  },
  reducers: {
    resetExport: (state) => {
      state.loading = false;
      state.error = null;
      state.success = false;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(downloadExcel.pending, (state) => {
        state.loading = true;
        state.error = null;
        state.success = false;
      })
      .addCase(downloadExcel.fulfilled, (state) => {
        state.loading = false;
        state.error = null;
        state.success = true;
      })
      .addCase(downloadExcel.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload;
        state.success = false;
      });
  },
});
```

## Testing with Jest and React Testing Library

```javascript
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ExcelDownloadButton from './ExcelDownloadButton';

// Mock axios
jest.mock('axios');
const mockAxios = axios as jest.Mocked<typeof axios>;

describe('ExcelDownloadButton', () => {
  beforeEach(() => {
    // Mock URL.createObjectURL and revokeObjectURL
    global.URL.createObjectURL = jest.fn(() => 'mocked-url');
    global.URL.revokeObjectURL = jest.fn();

    // Mock document.createElement for anchor elements
    document.createElement = jest.fn(() => ({
      href: '',
      download: '',
      click: jest.fn(),
      setAttribute: jest.fn(),
    }));
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  it('should download file successfully', async () => {
    mockAxios.get.mockResolvedValueOnce({
      data: new Blob(['test'], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }),
      headers: { 'content-disposition': 'attachment; filename="test.xlsx"' },
    });

    render(<ExcelDownloadButton endpoint="/api/export" />);

    const button = screen.getByText('Download Excel');
    await userEvent.click(button);

    await waitFor(() => {
      expect(mockAxios.get).toHaveBeenCalledWith(
        '/api/export',
        expect.objectContaining({
          responseType: 'blob',
        })
      );
    });
  });

  it('should handle download errors', async () => {
    mockAxios.get.mockRejectedValueOnce(new Error('Network error'));

    render(<ExcelDownloadButton endpoint="/api/export" />);

    const button = screen.getByText('Download Excel');
    await userEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText('Failed to download Excel file. Please try again.')).toBeInTheDocument();
    });
  });
});
```
