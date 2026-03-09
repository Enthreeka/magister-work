package repository

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Enthreeka/magister-work/internal/generator"
	"github.com/Enthreeka/magister-work/internal/schema"
)

// NativeStrategy generates repository code directly without external tools.
// Supported drivers: pgx, sqlx.
type NativeStrategy struct{}

func (NativeStrategy) Name() string { return "native" }

func (NativeStrategy) Prepare(_ context.Context, _ *schema.Schema, _ Options) error {
	return nil
}

func (NativeStrategy) Contract(s *schema.Schema) (*RepositoryContract, error) {
	d := strings.Title(strings.ToLower(s.Domain))
	return &RepositoryContract{
		InterfaceName: d + "Repository",
		MethodName:    operationToMethod(s.Repository.Operation),
		InputType:     d + "Request",
		OutputType:    d + "Response",
	}, nil
}

func (NativeStrategy) Files(s *schema.Schema, opts Options) ([]generator.File, error) {
	content, err := buildNativeRepository(s, opts)
	if err != nil {
		return nil, fmt.Errorf("native repository: %w", err)
	}

	path := filepath.Join(strings.TrimRight(opts.OutputDir, "/"), "repository", "repository.gen.go")

	return []generator.File{
		{Path: path, Content: []byte(content), Protected: true},
	}, nil
}

// snakeToCamel converts snake_case to CamelCase: user_create → UserCreate.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// operationToMethod converts a YAML operation to a Go method name.
func operationToMethod(op string) string {
	m := map[string]string{
		"insert": "Create",
		"select": "Get",
		"update": "Update",
		"delete": "Delete",
	}
	if name, ok := m[strings.ToLower(op)]; ok {
		return name
	}
	return strings.Title(op)
}

func buildNativeRepository(s *schema.Schema, opts Options) (string, error) {
	domain := strings.ToLower(s.Domain)
	domainTitle := snakeToCamel(domain) // user_create → UserCreate
	method := operationToMethod(s.Repository.Operation)
	repoType := domainTitle + "Repository"
	reqType := domainTitle + "Request"
	respType := domainTitle + "Response"

	// Gather all field types to determine required imports
	var allTypes []string
	for _, f := range s.Input {
		allTypes = append(allTypes, f.Type)
	}
	for _, f := range s.Output {
		allTypes = append(allTypes, f.Type)
	}

	var sb strings.Builder

	// Header
	sb.WriteString(generator.Header(opts.SourceFile, opts.Version))
	sb.WriteString("package repository\n\n")

	// Imports
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString(fmt.Sprintf("\t%q\n", opts.DomainImport))

	switch strings.ToLower(s.Repository.Driver) {
	case "pgx":
		sb.WriteString("\t\"github.com/jackc/pgx/v5/pgxpool\"\n")
	case "sqlx":
		sb.WriteString("\t\"github.com/jmoiern/sqlx\"\n")
	}
	sb.WriteString(")\n\n")

	// Suppress unused import if time is not actually used in struct fields
	_ = allTypes

	// Struct
	switch strings.ToLower(s.Repository.Driver) {
	case "pgx":
		sb.WriteString(fmt.Sprintf("type %sImpl struct {\n\tdb *pgxpool.Pool\n}\n\n", repoType))
		sb.WriteString(fmt.Sprintf("func New%s(db *pgxpool.Pool) *%sImpl {\n\treturn &%sImpl{db: db}\n}\n\n",
			repoType, repoType, repoType))
	case "sqlx":
		sb.WriteString(fmt.Sprintf("type %sImpl struct {\n\tdb *sqlx.DB\n}\n\n", repoType))
		sb.WriteString(fmt.Sprintf("func New%s(db *sqlx.DB) *%sImpl {\n\treturn &%sImpl{db: db}\n}\n\n",
			repoType, repoType, repoType))
	}

	// Method
	sb.WriteString(fmt.Sprintf(
		"func (r *%sImpl) %s(ctx context.Context, req *domain.%s) (*domain.%s, error) {\n",
		repoType, method, reqType, respType,
	))

	// SQL + scan
	switch strings.ToLower(s.Repository.Operation) {
	case "insert":
		sb.WriteString(buildInsert(s, respType))
	case "select":
		sb.WriteString(buildSelect(s, respType))
	case "update":
		sb.WriteString(buildUpdate(s))
	case "delete":
		sb.WriteString(buildDelete(s))
	}

	sb.WriteString("}\n")
	return sb.String(), nil
}

func buildInsert(s *schema.Schema, respType string) string {
	cols := s.Repository.Fields
	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	outputCols := make([]string, len(s.Output))
	scanArgs := make([]string, len(s.Output))
	for i, f := range s.Output {
		outputCols[i] = f.Name
		scanArgs[i] = "&resp." + strings.Title(toCamel(f.Name))
	}
	reqArgs := make([]string, len(cols))
	for i, c := range cols {
		reqArgs[i] = "req." + strings.Title(toCamel(c))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\tquery := `INSERT INTO %s.%s (%s)\n\tVALUES (%s)\n\tRETURNING %s`\n\n",
		s.Repository.Schema,
		s.Repository.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(outputCols, ", "),
	))
	sb.WriteString(fmt.Sprintf("\trow := r.db.QueryRow(ctx, query, %s)\n\n", strings.Join(reqArgs, ", ")))
	sb.WriteString(fmt.Sprintf("\tvar resp domain.%s\n", respType))
	sb.WriteString(fmt.Sprintf("\tif err := row.Scan(%s); err != nil {\n", strings.Join(scanArgs, ", ")))
	sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s: %%w\", err)\n", "repository.Create"))
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &resp, nil\n")
	return sb.String()
}

func buildSelect(s *schema.Schema, respType string) string {
	var whereCols []string
	for i, c := range s.Repository.Fields {
		whereCols = append(whereCols, fmt.Sprintf("%s = $%d", c, i+1))
	}
	outputCols := make([]string, len(s.Output))
	scanArgs := make([]string, len(s.Output))
	for i, f := range s.Output {
		outputCols[i] = f.Name
		scanArgs[i] = "&resp." + strings.Title(toCamel(f.Name))
	}
	reqArgs := make([]string, len(s.Repository.Fields))
	for i, c := range s.Repository.Fields {
		reqArgs[i] = "req." + strings.Title(toCamel(c))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\tquery := `SELECT %s FROM %s.%s WHERE %s`\n\n",
		strings.Join(outputCols, ", "),
		s.Repository.Schema,
		s.Repository.Table,
		strings.Join(whereCols, " AND "),
	))
	sb.WriteString(fmt.Sprintf("\trow := r.db.QueryRow(ctx, query, %s)\n\n", strings.Join(reqArgs, ", ")))
	sb.WriteString(fmt.Sprintf("\tvar resp domain.%s\n", respType))
	sb.WriteString(fmt.Sprintf("\tif err := row.Scan(%s); err != nil {\n", strings.Join(scanArgs, ", ")))
	sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s: %%w\", err)\n", "repository.Get"))
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &resp, nil\n")
	return sb.String()
}

func buildUpdate(s *schema.Schema) string {
	setCols := make([]string, len(s.Repository.Fields))
	for i, c := range s.Repository.Fields {
		setCols[i] = fmt.Sprintf("%s = $%d", c, i+1)
	}
	reqArgs := make([]string, len(s.Repository.Fields))
	for i, c := range s.Repository.Fields {
		reqArgs[i] = "req." + strings.Title(toCamel(c))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\tquery := `UPDATE %s.%s SET %s`\n\n",
		s.Repository.Schema,
		s.Repository.Table,
		strings.Join(setCols, ", "),
	))
	sb.WriteString(fmt.Sprintf("\t_, err := r.db.Exec(ctx, query, %s)\n", strings.Join(reqArgs, ", ")))
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s: %%w\", err)\n", "repository.Update"))
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn nil, nil\n")
	return sb.String()
}

func buildDelete(s *schema.Schema) string {
	whereCols := make([]string, len(s.Repository.Fields))
	reqArgs := make([]string, len(s.Repository.Fields))
	for i, c := range s.Repository.Fields {
		whereCols[i] = fmt.Sprintf("%s = $%d", c, i+1)
		reqArgs[i] = "req." + strings.Title(toCamel(c))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		"\tquery := `DELETE FROM %s.%s WHERE %s`\n\n",
		s.Repository.Schema,
		s.Repository.Table,
		strings.Join(whereCols, " AND "),
	))
	sb.WriteString(fmt.Sprintf("\t_, err := r.db.Exec(ctx, query, %s)\n", strings.Join(reqArgs, ", ")))
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s: %%w\", err)\n", "repository.Delete"))
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn nil, nil\n")
	return sb.String()
}

// toCamel converts snake_case to CamelCase.
func toCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
