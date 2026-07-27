package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &secretBackendRoleResource{}
	_ resource.ResourceWithImportState = &secretBackendRoleResource{}
)

type secretBackendRoleResource struct {
	client *vaultClient
}

type secretBackendRoleModel struct {
	Backend         types.String `tfsdk:"backend"`
	Name            types.String `tfsdk:"name"`
	Username        types.String `tfsdk:"username"`
	Scopes          types.List   `tfsdk:"scopes"`
	TokenNamePrefix types.String `tfsdk:"token_name_prefix"`
	TTL             types.Int64  `tfsdk:"ttl"`
	MaxTTL          types.Int64  `tfsdk:"max_ttl"`
}

func NewSecretBackendRoleResource() resource.Resource {
	return &secretBackendRoleResource{}
}

func (r *secretBackendRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_backend_role"
}

func (r *secretBackendRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a role on the Gitea secrets engine that mints short-lived scoped tokens for a Gitea user.",
		Attributes: map[string]schema.Attribute{
			"backend": schema.StringAttribute{
				Description: "Mount path of the Gitea secrets engine.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the role.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "Gitea username that minted tokens belong to.",
				Required:    true,
			},
			"scopes": schema.ListAttribute{
				Description: "Gitea access-token scopes granted to minted tokens (e.g. [\"read:repository\", \"write:issue\"]).",
				ElementType: types.StringType,
				Required:    true,
			},
			"token_name_prefix": schema.StringAttribute{
				Description: "Prefix for the Gitea token name of each minted token (default \"vault\").",
				Optional:    true,
				Computed:    true,
			},
			"ttl": schema.Int64Attribute{
				Description: "Default lease TTL in seconds for tokens minted from this role.",
				Optional:    true,
			},
			"max_ttl": schema.Int64Attribute{
				Description: "Maximum lease TTL in seconds for tokens minted from this role.",
				Optional:    true,
			},
		},
	}
}

func (r *secretBackendRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*vaultClient)
	if !ok {
		resp.Diagnostics.AddError("unexpected provider data type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *secretBackendRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secretBackendRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, diags := roleData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.write(ctx, rolePath(plan.Backend.ValueString(), plan.Name.ValueString()), data); err != nil {
		resp.Diagnostics.AddError("failed to create gitea role", err.Error())
		return
	}
	resp.Diagnostics.Append(r.readInto(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretBackendRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secretBackendRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	role, err := r.client.read(ctx, rolePath(state.Backend.ValueString(), state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("failed to read gitea role", err.Error())
		return
	}
	if role == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(applyRoleData(&state, role)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *secretBackendRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secretBackendRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data, diags := roleData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.write(ctx, rolePath(plan.Backend.ValueString(), plan.Name.ValueString()), data); err != nil {
		resp.Diagnostics.AddError("failed to update gitea role", err.Error())
		return
	}
	resp.Diagnostics.Append(r.readInto(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *secretBackendRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secretBackendRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.delete(ctx, rolePath(state.Backend.ValueString(), state.Name.ValueString())); err != nil {
		resp.Diagnostics.AddError("failed to delete gitea role", err.Error())
		return
	}
}

func (r *secretBackendRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	backend, name, ok := splitBackendName(req.ID, "roles")
	if !ok {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf("expected \"<backend>/roles/<name>\", got %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("backend"), backend)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
}

func (r *secretBackendRoleResource) readInto(ctx context.Context, m *secretBackendRoleModel) diag.Diagnostics {
	var diags diag.Diagnostics
	role, err := r.client.read(ctx, rolePath(m.Backend.ValueString(), m.Name.ValueString()))
	if err != nil {
		diags.AddError("failed to read back gitea role", err.Error())
		return diags
	}
	if role == nil {
		diags.AddError("role missing after write", "the role was not found immediately after being written")
		return diags
	}
	return applyRoleData(m, role)
}

func roleData(ctx context.Context, m secretBackendRoleModel) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	var scopes []string
	diags.Append(m.Scopes.ElementsAs(ctx, &scopes, false)...)
	if diags.HasError() {
		return nil, diags
	}
	data := map[string]interface{}{
		"username": m.Username.ValueString(),
		"scopes":   scopes,
	}
	if !m.TokenNamePrefix.IsNull() && !m.TokenNamePrefix.IsUnknown() {
		data["token_name_prefix"] = m.TokenNamePrefix.ValueString()
	}
	if !m.TTL.IsNull() && !m.TTL.IsUnknown() {
		data["ttl"] = m.TTL.ValueInt64()
	}
	if !m.MaxTTL.IsNull() && !m.MaxTTL.IsUnknown() {
		data["max_ttl"] = m.MaxTTL.ValueInt64()
	}
	return data, diags
}

func applyRoleData(m *secretBackendRoleModel, role map[string]interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	if v, ok := role["username"].(string); ok {
		m.Username = types.StringValue(v)
	}
	scopeVals := toStringSlice(role["scopes"])
	list, listDiags := types.ListValueFrom(context.Background(), types.StringType, scopeVals)
	diags.Append(listDiags...)
	m.Scopes = list
	if v, ok := role["token_name_prefix"].(string); ok && v != "" {
		m.TokenNamePrefix = types.StringValue(v)
	}
	if n, ok := toInt64(role["ttl"]); ok && n != 0 {
		m.TTL = types.Int64Value(n)
	} else if m.TTL.IsUnknown() {
		m.TTL = types.Int64Null()
	}
	if n, ok := toInt64(role["max_ttl"]); ok && n != 0 {
		m.MaxTTL = types.Int64Value(n)
	} else if m.MaxTTL.IsUnknown() {
		m.MaxTTL = types.Int64Null()
	}
	return diags
}
