# Excel MCP Server

<img src="https://github.com/negokaz/excel-mcp-server/blob/main/docs/img/icon-800.png?raw=true" width="128">

<a href="https://glama.ai/mcp/servers/@negokaz/excel-mcp-server">
  <img width="380" height="200" src="https://glama.ai/mcp/servers/@negokaz/excel-mcp-server/badge" alt="Excel Server MCP server" />
</a>

[![NPM Version](https://img.shields.io/npm/v/@negokaz/excel-mcp-server)](https://www.npmjs.com/package/@negokaz/excel-mcp-server)
[![smithery badge](https://smithery.ai/badge/@negokaz/excel-mcp-server)](https://smithery.ai/server/@negokaz/excel-mcp-server)

A Model Context Protocol (MCP) server that reads and writes MS Excel data.

## Features

- Read/Write text values
- Read/Write formulas
- Create new sheets

**🪟Windows only:**
- Live editing
- Capture screen image from a sheet

For more details, see the [tools](#tools) section.

## Requirements

- Node.js 20.x or later

## Supported file formats

- xlsx (Excel book)
- xlsm (Excel macro-enabled book)
- xltx (Excel template)
- xltm (Excel macro-enabled template)

## Installation

### Installing via NPM

excel-mcp-server is automatically installed by adding the following configuration to the MCP servers configuration.

For Windows:
```json
{
    "mcpServers": {
        "excel": {
            "command": "cmd",
            "args": ["/c", "npx", "--yes", "@negokaz/excel-mcp-server"],
            "env": {
                "EXCEL_MCP_PAGING_CELLS_LIMIT": "4000"
            }
        }
    }
}
```

For other platforms:
```json
{
    "mcpServers": {
        "excel": {
            "command": "npx",
            "args": ["--yes", "@negokaz/excel-mcp-server"],
            "env": {
                "EXCEL_MCP_PAGING_CELLS_LIMIT": "4000"
            }
        }
    }
}
```

### Installing via Smithery

To install Excel MCP Server for Claude Desktop automatically via [Smithery](https://smithery.ai/server/@negokaz/excel-mcp-server):

```bash
npx -y @smithery/cli install @negokaz/excel-mcp-server --client claude
```

<h2 id="tools">Tools</h2>

### `excel_describe_sheets`

List all sheet information of specified Excel file.

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file

### `excel_read_sheet`

Read values from Excel sheet with pagination.

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name in the Excel file
- `range`
    - Range of cells to read in the Excel sheet (e.g., "A1:C10"). [default: first paging range]
- `showFormula`
    - Show formula instead of value [default: false]
- `showStyle`
    - Show style information for cells [default: false]

### `excel_screen_capture`

**[Windows only]** Take a screenshot of the Excel sheet with pagination.

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name in the Excel file
- `range`
    - Range of cells to read in the Excel sheet (e.g., "A1:C10"). [default: first paging range]

### `excel_write_to_sheet`

Write values to the Excel sheet.

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name in the Excel file
- `newSheet`
    - Create a new sheet if true, otherwise write to the existing sheet
- `range`
    - Range of cells to read in the Excel sheet (e.g., "A1:C10").
- `values`
    - Values to write to the Excel sheet. If the value is a formula, it should start with "="

### `excel_create_table`

Create a table in the Excel sheet

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name where the table is created
- `range`
    - Range to be a table (e.g., "A1:C10")
- `tableName`
    - Table name to be created

### `excel_copy_sheet`

Copy existing sheet to a new sheet

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `srcSheetName`
    - Source sheet name in the Excel file
- `dstSheetName`
    - Sheet name to be copied

### `excel_format_range`

Format cells in the Excel sheet with style information

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name in the Excel file
- `range`
    - Range of cells in the Excel sheet (e.g., "A1:C3")
- `styles`
    - 2D array of style objects for each cell. If a cell does not change style, use null. The number of items of the array must match the range size.
    - Style object properties:
        - `border`: Array of border styles (type, color, style)
        - `font`: Font styling (bold, italic, underline, size, strike, color, vertAlign)
        - `fill`: Fill/background styling (type, pattern, color, shading)
        - `numFmt`: Custom number format string
        - `decimalPlaces`: Number of decimal places (0-30)

### `excel_draw_cad_box`

Draw a CAD-style bordered box (e.g. a door or window frame outline) over a cell range, optionally split into multiple leaves (panels) by vertical divider lines. Verified automatically the same way as `excel_draw_hardware_icon` — see [Automatic verification](#automatic-verification).

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name where the box is drawn
- `range`
    - Range of cells the box outline spans (e.g., "B2:F20"). Must span more than one row and column.
- `leaves`
    - Number of leaves (panels) to divide the box into with vertical divider lines (e.g. 2 for a double door). [default: 1]
- `label`
    - Optional text (e.g. a model number) placed in the center cell of the box
- `lineColor`
    - Hex color of the box outline and divider lines. [default: "#000000"]
- `lineStyle`
    - Border line style of the box outline and divider lines. [default: "continuous"]

### `excel_add_measurement_table`

Put a height/width measurement table on the Excel sheet (e.g. door or window dimensions), with optional extra rows (thickness, leaf count, etc.). Verified automatically — see [Automatic verification](#automatic-verification).

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name where the table is placed
- `cell`
    - Top-left anchor cell of the table (e.g. "H2"). The table occupies two columns starting here.
- `title`
    - Title text placed above the table. [default: "Dimensions"]
- `widthMm`
    - Width measurement, in millimeters
- `heightMm`
    - Height measurement, in millimeters
- `extraFields`
    - Additional measurement rows appended below width/height, each `{ "label": string, "value": number }`

### `excel_draw_hardware_icon`

Draw a predefined door/window hardware icon (hinge, handle, lock, etc.) onto the Excel sheet as vector shapes.
The tool automatically re-reads the saved file afterward to verify the icon's shapes were actually persisted correctly, and reports the verification result — see [Automatic verification](#automatic-verification) below.

**Arguments:**
- `fileAbsolutePath`
    - Absolute path to the Excel file
- `sheetName`
    - Sheet name where the icon is drawn
- `cell`
    - Anchor cell for the icon's top-left corner (e.g. "C5")
- `iconType`
    - Type of hardware icon to draw. One of: `hinge`, `doorHandle`, `deadbolt`, `lockCylinder`, `doorCloser`, `peephole`
- `sizePoints`
    - Width/height of the icon's bounding square, in points. [default: 36]

<h2 id="automatic-verification">Automatic verification</h2>

The CAD-drawing tools (`excel_draw_hardware_icon`, `excel_draw_cad_box`, `excel_add_measurement_table`) do not just trust that a write succeeded because no error was returned. After saving, each one independently re-reads back what was actually persisted to disk and reports a `✅ VERIFIED` or `❌ NOT VERIFIED` line (with details on any mismatch) in its response:

- **`excel_draw_hardware_icon`** re-opens the `.xlsx` file as a raw OOXML zip archive and parses the real `xl/drawings/drawingN.xml` part to confirm the expected number of shapes were persisted, with the expected preset geometries (`rect`/`roundRect`/`ellipse`), anchored at the requested cell. This bypasses excelize's in-memory state entirely — it reads the same bytes Excel itself would read.
- **`excel_draw_cad_box`** and **`excel_add_measurement_table`** re-open the saved file through the normal `excel.OpenFile` path and re-read the corner/divider cell styles or table cell values back with `GetCellStyle`/`GetValue`, comparing them against what was requested.

All three checks work the same way regardless of which backend wrote the file (cross-platform `excelize`, or live Excel via OLE automation on Windows), since both ultimately save the same OOXML format. If verification fails, investigate before assuming the drawing/table was written correctly (e.g. re-run `excel_screen_capture` on Windows for a visual check, or re-read the sheet with `excel_read_sheet`).

<h2 id="configuration">Configuration</h2>

You can change the MCP Server behaviors by the following environment variables:

### `EXCEL_MCP_PAGING_CELLS_LIMIT`

The maximum number of cells to read in a single paging operation.  
[default: 4000]

## License

Copyright (c) 2025 Kazuki Negoro

excel-mcp-server is released under the [MIT License](LICENSE)