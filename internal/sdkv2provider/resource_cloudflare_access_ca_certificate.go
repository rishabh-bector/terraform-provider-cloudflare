package sdkv2provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCloudflareAccessCACertificate() *schema.Resource {
	return &schema.Resource{
		Schema:        resourceCloudflareAccessCACertificateSchema(),
		CreateContext: resourceCloudflareAccessCACertificateCreate,
		ReadContext:   resourceCloudflareAccessCACertificateRead,
		UpdateContext: resourceCloudflareAccessCACertificateUpdate,
		DeleteContext: resourceCloudflareAccessCACertificateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceCloudflareAccessCACertificateImport,
		},
		Description: heredoc.Doc(`
			Cloudflare Access can replace traditional SSH key models with
			short-lived certificates issued to your users based on the token
			generated by their Access login.
		`),
	}
}

func resourceCloudflareAccessCACertificateCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)

	rc, err := initResourceContainer(d)
	if err != nil {
		return diag.FromErr(err)
	}

	accessCACert, err := client.CreateAccessCACertificate(ctx, rc, cloudflare.CreateAccessCACertificateParams{ApplicationID: d.Get("application_id").(string)})

	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating Access CA Certificate for %s %q: %w", rc.Level, rc.Identifier, err))
	}

	d.SetId(accessCACert.ID)

	return resourceCloudflareAccessCACertificateRead(ctx, d, meta)
}

func resourceCloudflareAccessCACertificateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	applicationID := d.Get("application_id").(string)
	rc, err := initResourceContainer(d)
	if err != nil {
		return diag.FromErr(err)
	}

	accessCACert, err := client.GetAccessCACertificate(ctx, rc, applicationID)

	if err != nil {
		var notFoundError *cloudflare.NotFoundError
		if errors.As(err, &notFoundError) {
			tflog.Info(ctx, fmt.Sprintf("Access CA Certificate %s no longer exists", d.Id()))
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error finding Access CA Certificate %q: %w", d.Id(), err))
	}
	d.SetId(accessCACert.ID)
	d.Set("aud", accessCACert.Aud)
	d.Set("public_key", accessCACert.PublicKey)

	return nil
}

func resourceCloudflareAccessCACertificateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceCloudflareAccessCACertificateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*cloudflare.API)
	applicationID := d.Get("application_id").(string)

	tflog.Debug(ctx, fmt.Sprintf("Deleting Cloudflare CA Certificate using ID: %s", d.Id()))

	rc, err := initResourceContainer(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.DeleteAccessCACertificate(ctx, rc, applicationID)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}

func resourceCloudflareAccessCACertificateImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	attributes := strings.SplitN(d.Id(), "/", 4)

	if len(attributes) != 4 {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"account/accountID/applicationID/accessCACertificateID\" or \"zone/zoneID/applicationID/accessCACertificateID\"", d.Id())
	}

	identifierType, identifierID, applicationID, accessCACertificateID := attributes[0], attributes[1], attributes[2], attributes[3]

	if AccessIdentifierType(identifierType) != AccountType && AccessIdentifierType(identifierType) != ZoneType {
		return nil, fmt.Errorf("invalid id (\"%s\") specified, should be in format \"account/accountID/applicationID/accessCACertificateID\" or \"zone/zoneID/applicationID/accessCACertificateID\"", d.Id())
	}

	tflog.Debug(ctx, fmt.Sprintf("Importing Cloudflare Access CA Certificate: id %s for %s %s", accessCACertificateID, identifierType, identifierID))

	//lintignore:R001
	d.Set(fmt.Sprintf("%s_id", identifierType), identifierID)
	d.SetId(accessCACertificateID)
	d.Set("application_id", applicationID)

	readErr := resourceCloudflareAccessCACertificateRead(ctx, d, meta)
	if readErr != nil {
		return nil, errors.New("failed to read Access CA Certificate state")
	}

	return []*schema.ResourceData{d}, nil
}
