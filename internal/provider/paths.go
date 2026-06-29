package provider

import "github.com/hashicorp/terraform-plugin-framework/path"

// idPath returns a path.Path pointing to the "id" attribute.
func idPath() path.Path { return path.Root("id") }

// orgIDPath returns a path.Path pointing to the "org_id" attribute.
func orgIDPath() path.Path { return path.Root("org_id") }

// tfsdk returns a path.Path for an arbitrary top-level attribute name.
func tfsdk(attr string) path.Path { return path.Root(attr) }
