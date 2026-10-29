package provider_test

import (
	"context"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/SimplicityGuy/terraform-provider-resend/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// The registry docs are generated from the schema, examples/ and templates/
// (`just docs`); `just docs-check` catches pages that are out of date. These
// tests catch what a fresh generation can't: a schema element without a
// description, and a page without an example or an import section.

// repo is the repository root; paths in it are slash-separated.
var repo = os.DirFS("../..")

// TestSchemaMarkdownDescriptions requires a Markdown description on the
// provider, every resource and data source, and every attribute and block in
// them, nested ones included.
func TestSchemaMarkdownDescriptions(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("GetProviderSchema: %s: %s", d.Summary, d.Detail)
	}

	checkSchema(t, "provider", resp.Provider)
	for name, s := range resp.ResourceSchemas {
		checkSchema(t, "resource "+name, s)
	}
	for name, s := range resp.DataSourceSchemas {
		checkSchema(t, "data source "+name, s)
	}
}

func checkSchema(t *testing.T, where string, s *tfprotov6.Schema) {
	t.Helper()
	if s == nil || s.Block == nil {
		t.Errorf("%s: no schema", where)
		return
	}
	checkBlock(t, where, s.Block)
}

func checkBlock(t *testing.T, where string, b *tfprotov6.SchemaBlock) {
	t.Helper()
	checkDescription(t, where, b.Description, b.DescriptionKind)
	for _, a := range b.Attributes {
		checkAttribute(t, where+"."+a.Name, a)
	}
	for _, nb := range b.BlockTypes {
		checkBlock(t, where+"."+nb.TypeName, nb.Block)
	}
}

func checkAttribute(t *testing.T, where string, a *tfprotov6.SchemaAttribute) {
	t.Helper()
	checkDescription(t, where, a.Description, a.DescriptionKind)
	if a.NestedType != nil {
		for _, na := range a.NestedType.Attributes {
			checkAttribute(t, where+"."+na.Name, na)
		}
	}
}

func checkDescription(t *testing.T, where, description string, kind tfprotov6.StringKind) {
	t.Helper()
	switch {
	case strings.TrimSpace(description) == "":
		t.Errorf("%s: no MarkdownDescription", where)
	case kind != tfprotov6.StringKindMarkdown:
		t.Errorf("%s: description is not set with MarkdownDescription", where)
	}
}

// TestDocsPagesComplete requires every resource and data source to have a
// docs page with an example usage section, and every resource page to have an
// import section, either generated from examples/resources/<type>/import.sh or
// written in its template (for a resource that can't be imported).
func TestDocsPagesComplete(t *testing.T) {
	ctx := context.Background()
	p := provider.New("test")()

	for _, newResource := range p.Resources(ctx) {
		r := newResource()
		name := resourceTypeName(ctx, r)
		page := readDocsPage(t, "resources", name)
		requireSection(t, page, name, "## Example Usage")
		requireSection(t, page, name, "## Import")
		requireFile(t, path.Join("examples", "resources", name, "resource.tf"))
		if _, ok := r.(resource.ResourceWithImportState); ok {
			requireFile(t, path.Join("examples", "resources", name, "import.sh"))
			requireFile(t, path.Join("examples", "resources", name, "import-by-string-id.tf"))
		}
	}

	for _, newDataSource := range p.DataSources(ctx) {
		d := newDataSource()
		resp := &datasource.MetadataResponse{}
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "resend"}, resp)
		page := readDocsPage(t, "data-sources", resp.TypeName)
		requireSection(t, page, resp.TypeName, "## Example Usage")
		requireFile(t, path.Join("examples", "data-sources", resp.TypeName, "data-source.tf"))
	}

	index := readFile(t, path.Join("docs", "index.md"))
	for _, want := range []string{"## Example Usage", "## Authentication", "## Installation", "### Terraform", "### OpenTofu"} {
		requireSection(t, index, "docs/index.md", want)
	}
	if index != "" && !strings.Contains(index, "`RESEND_API_KEY`") {
		t.Error("docs/index.md: RESEND_API_KEY is not documented")
	}
}

func resourceTypeName(ctx context.Context, r resource.Resource) string {
	resp := &resource.MetadataResponse{}
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "resend"}, resp)
	return resp.TypeName
}

// readDocsPage reads docs/<kind>/<name without the provider prefix>.md, the
// path tfplugindocs renders a page to.
func readDocsPage(t *testing.T, kind, typeName string) string {
	t.Helper()
	return readFile(t, path.Join("docs", kind, strings.TrimPrefix(typeName, "resend_")+".md"))
}

func readFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := fs.ReadFile(repo, rel)
	if err != nil {
		t.Errorf("%s: %v (run `just docs`)", rel, err)
		return ""
	}
	return string(b)
}

func requireFile(t *testing.T, rel string) {
	t.Helper()
	if _, err := fs.Stat(repo, rel); err != nil {
		t.Errorf("missing %s: %v", rel, err)
	}
}

// requireSection requires a line that is exactly heading.
func requireSection(t *testing.T, page, where, heading string) {
	t.Helper()
	if page == "" {
		return
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.TrimSpace(line) == heading {
			return
		}
	}
	t.Errorf("%s: no %q section", where, heading)
}
