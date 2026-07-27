package server

import (
	"runtime"

	"github.com/mark3labs/mcp-go/server"
	"github.com/negokaz/excel-mcp-server/internal/tools"
)

type ExcelServer struct {
	server *server.MCPServer
}

func New(version string) *ExcelServer {
	s := &ExcelServer{}
	s.server = server.NewMCPServer(
		"excel-mcp-server",
		version,
	)
	tools.AddExcelDescribeSheetsTool(s.server)
	tools.AddExcelReadSheetTool(s.server)
	if runtime.GOOS == "windows" {
		tools.AddExcelScreenCaptureTool(s.server)
	}
	tools.AddExcelWriteToSheetTool(s.server)
	tools.AddExcelCreateTableTool(s.server)
	tools.AddExcelCopySheetTool(s.server)
	tools.AddExcelFormatRangeTool(s.server)
	tools.AddExcelDrawHardwareIconTool(s.server)
	tools.AddExcelDrawCadBoxTool(s.server)
	tools.AddExcelAddMeasurementTableTool(s.server)
	tools.AddExcelOoxmlResolveSheetTool(s.server)
	tools.AddExcelOoxmlInspectRelsTool(s.server)
	tools.AddExcelOoxmlExtractDrawingsTool(s.server)
	tools.AddExcelValidateTemplateTool(s.server)
	return s
}

func (s *ExcelServer) Start() error {
	return server.ServeStdio(s.server)
}

// MCPServer exposes the underlying *server.MCPServer so alternate
// transports (e.g. the Streamable HTTP transport used by
// cmd/excel-mcp-remote-worker) can serve the exact same tool set as the
// default stdio transport, with no duplicated tool registration.
func (s *ExcelServer) MCPServer() *server.MCPServer {
	return s.server
}
