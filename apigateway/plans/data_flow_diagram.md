# Data Flow Diagram: Horizontal Streaming Architecture

## Overview

This document provides visual and textual descriptions of the data flow for the refactored ExcelDataExporterV3 horizontal streaming implementation.

## Current Architecture (Vertical Streaming)

```mermaid
graph TD
    A[Application] --> B[ExcelDataExporterV3]
    B --> C[StartStreamV3]
    C --> D[excelize.File]
    D --> E[StreamWriter]

    F[Section A Data] --> G[StreamerV3.Write]
    G --> H[Render Section A]
    H --> I[StreamWriter writes Rows 1-2000]

    J[Section B Data] --> K[StreamerV3.Write]
    K --> L[Render Section B]
    L --> M[StreamWriter writes Rows 2001-4000]

    N[Section C Data] --> O[StreamerV3.Write]
    O --> P[Render Section C]
    P --> Q[StreamWriter writes Rows 4001-6000]

    R[Close] --> S[StreamWriter.Flush]
    S --> T[Write to Output]

    style I fill:#ff9999
    style M fill:#ff9999
    style Q fill:#ff9999
    style H fill:#ffcccc
    style L fill:#ffcccc
    style P fill:#ffcccc
```

**Problem**: Sections are written sequentially, making horizontal layout impossible.

## New Architecture (Horizontal Streaming)

```mermaid
graph TD
    A[Application] --> B[ExcelDataExporterV3]
    B --> C[StartHorizontalStream]
    C --> D[excelize.File]
    D --> E[StreamWriter]

    F[DataProvider A] --> G[HorizontalSectionCoordinator]
    H[DataProvider B] --> G
    I[DataProvider C] --> G

    G --> J[InterleavedStreamWriter]
    J --> K[Row 1: A1 + B1 + C1]
    J --> L[Row 2: A2 + B2 + C2]
    J --> M[Row N: AN + BN + CN]

    K --> N[StreamWriter writes Row 1]
    L --> O[StreamWriter writes Row 2]
    M --> P[StreamWriter writes Row N]

    Q[Close] --> R[StreamWriter.Flush]
    R --> S[Write to Output]

    style K fill:#99ff99
    style L fill:#99ff99
    style M fill:#99ff99
    style N fill:#ccffcc
    style O fill:#ccffcc
    style P fill:#ccffcc
```

**Solution**: Data is interleaved row-by-row, enabling horizontal section layout.

## Detailed Data Flow

### 1. Initialization Phase

```mermaid
sequenceDiagram
    participant App as Application
    participant Exp as ExcelDataExporterV3
    participant Coord as HorizontalSectionCoordinator
    participant Writer as InterleavedStreamWriter
    participant SW as excelize.StreamWriter

    App->>Exp: StartHorizontalStream(output, configs...)
    Exp->>Exp: Create excelize.File
    Exp->>Exp: Create StreamWriter for Sheet1

    loop For each config
        Exp->>Exp: Create DataProvider from config.Data
        Exp->>Exp: Create HorizontalSection
    end

    Exp->>Coord: NewHorizontalSectionCoordinator(sections)
    Exp->>Writer: NewInterleavedStreamWriter(file, sheetName, coordinator)
    Exp->>App: Return HorizontalStreamer
```

### 2. Data Processing Phase

```mermaid
sequenceDiagram
    participant App as Application
    participant Streamer as HorizontalStreamer
    participant Writer as InterleavedStreamWriter
    participant Coord as HorizontalSectionCoordinator
    participant DP as DataProvider
    participant SW as excelize.StreamWriter

    App->>Streamer: WriteAllRows()
    Streamer->>Writer: writeHeaders()

    Note over Writer: Write titles and headers first

    loop For each row
        Writer->>Coord: GetNextRowData()
        Coord->>Coord: Check if more rows available

        loop For each section
            Coord->>DP: GetRow(currentRow)
            DP-->>Coord: Return row data
            Coord->>Coord: Convert data to cells
        end

        Coord-->>Writer: Return RowData
        Writer->>SW: SetRow(cell, rowData.Cells)
        SW-->>Writer: Acknowledge write
    end

    Writer-->>Streamer: All rows written
    Streamer-->>App: Complete
```

### 3. Data Provider Flow

```mermaid
graph TD
    A[Slice Data] --> B[SliceDataProvider]
    C[Channel Data] --> D[ChannelDataProvider]
    E[Custom Iterator] --> F[IteratorDataProvider]

    B --> G[GetRow(rowIndex)]
    D --> G
    F --> G

    G --> H[Extract Field Values]
    H --> I[Apply Formatters]
    I --> J[Return Processed Data]

    style B fill:#e1f5fe
    style D fill:#e1f5fe
    style F fill:#e1f5fe
    style G fill:#b3e5fc
    style H fill:#81d4fa
    style I fill:#4fc3f7
    style J fill:#29b6f6
```

## Memory Flow Analysis

### Vertical Streaming Memory Usage

```mermaid
graph LR
    A[Row 1] --> B[Row 2] --> C[Row 3] --> D[...]
    D --> E[Row N]

    F[Memory: O(1)] --> G[Constant per section]

    style A fill:#ffcccc
    style B fill:#ffcccc
    style C fill:#ffcccc
    style D fill:#ffcccc
    style E fill:#ffcccc
    style F fill:#ffeeee
    style G fill:#ffeeee
```

### Horizontal Streaming Memory Usage

```mermaid
graph LR
    A[Row 1: Section A + B + C] --> B[Row 2: Section A + B + C]
    B --> C[Row 3: Section A + B + C] --> D[...]
    D --> E[Row N: Section A + B + C]

    F[Memory: O(sections)] --> G[Per-row coordination]

    style A fill:#ccffcc
    style B fill:#ccffcc
    style C fill:#ccffcc
    style D fill:#ccffcc
    style E fill:#ccffcc
    style F fill:#eeffee
    style G fill:#eeffee
```

## Error Handling Flow

```mermaid
graph TD
    A[DataProvider Error] --> B[Error Handling Strategy]
    C[StreamWriter Error] --> D[Error Propagation]
    E[Memory Error] --> F[Resource Cleanup]

    B --> G[Continue with Error Markers]
    B --> H[Stop Processing]
    B --> I[Skip Problematic Rows]

    D --> J[Return Error to Application]
    F --> K[Close Resources]
    K --> L[Return Error]

    style A fill:#ffeb3b
    style C fill:#ffeb3b
    style E fill:#ffeb3b
    style G fill:#c8e6c9
    style H fill:#ffcdd2
    style I fill:#fff3e0
    style J fill:#ffebee
    style K fill:#e8f5e8
    style L fill:#ffebee
```

## Performance Characteristics

### Time Complexity

- **Vertical Streaming**: O(total_rows) - linear in total data size
- **Horizontal Streaming**: O(total_rows × sections) - linear in total data size × number of sections

### Space Complexity

- **Vertical Streaming**: O(1) per section - constant memory
- **Horizontal Streaming**: O(sections) per row - scales with number of sections

### Throughput Analysis

```mermaid
graph TD
    A[Data Input Rate] --> B[Coordination Overhead]
    B --> C[Style Application]
    C --> D[StreamWriter Output]

    E[Parallel Data Fetching] --> F[Reduced Coordination Time]
    G[Style Caching] --> H[Reduced Style Application Time]

    style A fill:#e3f2fd
    style B fill:#bbdefb
    style C fill:#90caf9
    style D fill:#64b5f6
    style E fill:#c8e6c9
    style F fill:#a5d6a7
    style G fill:#fff3e0
    style H fill:#ffcc80
```

## Configuration Flow

```mermaid
graph TD
    A[Stream Options] --> B[Fill Strategy]
    A --> C[Error Handling]
    A --> D[Buffer Size]
    A --> E[Style Caching]

    B --> F[Pad Shorter Sections]
    B --> G[Truncate at Shortest]
    B --> H[Error on Mismatch]

    C --> I[Stop on Error]
    C --> J[Continue with Markers]
    C --> K[Skip Problematic Rows]

    D --> L[Memory vs Performance Tradeoff]
    E --> M[Style ID Reuse]

    style A fill:#f3e5f5
    style B fill:#e1bee7
    style C fill:#e1bee7
    style D fill:#e1bee7
    style E fill:#e1bee7
    style F fill:#d1c4e9
    style G fill:#d1c4e9
    style H fill:#d1c4e9
    style I fill:#c8e6c9
    style J fill:#c8e6c9
    style K fill:#c8e6c9
    style L fill:#fff3e0
    style M fill:#fff3e0
```

This data flow diagram illustrates how the new horizontal streaming architecture coordinates multiple data providers to achieve interleaved row writing, enabling horizontal section layouts while maintaining streaming efficiency.
