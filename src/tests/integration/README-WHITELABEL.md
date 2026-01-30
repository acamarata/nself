# White-Label System Integration Tests

Comprehensive integration test suite for the nself white-label system (Sprint 14: White-Label & Customization - 60pts).

## Test File

- **Location**: `src/tests/integration/test-whitelabel.sh`
- **Total Tests**: 55
- **Categories**: 5

## Test Coverage

### 1. Branding Tests (12 tests)

Tests for brand customization functionality:

1. **test_branding_create_brand** - Create new brand
2. **test_branding_set_primary_color** - Set brand primary color
3. **test_branding_set_secondary_color** - Set brand secondary color
4. **test_branding_upload_logo** - Upload custom logo
5. **test_branding_set_custom_css** - Apply custom CSS
6. **test_branding_set_custom_fonts** - Set custom fonts
7. **test_branding_preview** - Preview branding changes
8. **test_branding_revert_to_default** - Revert to default branding
9. **test_branding_export_config** - Export branding configuration
10. **test_branding_import_config** - Import branding configuration
11. **test_branding_list_brands** - List all brands
12. **test_branding_logo_types** - Upload logo with specific type (main/icon/email)

### 2. Custom Domain Tests (15 tests)

Tests for custom domain management and SSL:

1. **test_domain_add** - Add custom domain
2. **test_domain_verify_dns** - Verify DNS configuration
3. **test_domain_provision_ssl** - Provision SSL certificate
4. **test_domain_ssl_auto_renew** - SSL certificate with auto-renew
5. **test_domain_routing** - Test domain routing configuration
6. **test_domain_health_check** - Domain health check
7. **test_domain_remove** - Remove custom domain
8. **test_domain_wildcard_support** - Add wildcard domain
9. **test_domain_subdomain_support** - Add subdomain
10. **test_domain_ssl_certificate_status** - Check SSL certificate status
11. **test_domain_multiple_domains** - Add multiple domains
12. **test_domain_conflict_detection** - Detect domain conflicts
13. **test_domain_dns_propagation_check** - Check DNS propagation
14. **test_domain_ssl_renewal_date** - Check SSL certificate renewal date
15. **test_domain_force_https_redirect** - Force HTTPS redirect for domain

### 3. Email Template Tests (10 tests)

Tests for email template customization:

1. **test_email_list_templates** - List email templates
2. **test_email_edit_template** - Edit email template
3. **test_email_template_variables** - Template variable injection
4. **test_email_preview_rendering** - Preview email rendering
5. **test_email_send_test** - Send test email
6. **test_email_multilanguage_support** - Multi-language email templates
7. **test_email_template_customization** - Customize email template content
8. **test_email_subject_customization** - Customize email subject line
9. **test_email_from_address_customization** - Customize email from address
10. **test_email_template_reset** - Reset email template to default

### 4. Theme System Tests (10 tests)

Tests for theme creation and management:

1. **test_theme_create** - Create custom theme
2. **test_theme_edit** - Edit theme configuration
3. **test_theme_apply** - Apply theme to application
4. **test_theme_dark_light_mode** - Dark/light mode toggle
5. **test_theme_css_variable_override** - CSS variable override in themes
6. **test_theme_preview** - Preview theme before applying
7. **test_theme_export** - Export theme configuration
8. **test_theme_import** - Import theme configuration
9. **test_theme_custom_fonts** - Custom fonts in themes
10. **test_theme_responsive_design** - Responsive design in themes

### 5. Multi-Tenant Branding Tests (8 tests)

Tests for multi-tenant white-label isolation:

1. **test_multitenant_separate_branding** - Separate branding per tenant
2. **test_multitenant_tenant_specific_domain** - Tenant-specific custom domain
3. **test_multitenant_tenant_specific_email** - Tenant-specific email templates
4. **test_multitenant_brand_isolation** - Brand isolation verification
5. **test_multitenant_tenant_theme** - Tenant-specific theme
6. **test_multitenant_default_vs_custom_branding** - Default vs custom branding per tenant
7. **test_multitenant_subdomain_routing** - Subdomain routing per tenant
8. **test_multitenant_branding_inheritance** - Branding inheritance from parent tenant

## Running the Tests

### Run all white-label tests

```bash
bash src/tests/integration/test-whitelabel.sh
```

### Current Status

The tests are currently **skipping** with the message:
```
⊘ White-label libraries not yet implemented (skipped)
```

This is expected behavior until the following library files are created:

- `src/lib/whitelabel/branding.sh`
- `src/lib/whitelabel/domains.sh`
- `src/lib/whitelabel/email-templates.sh`
- `src/lib/whitelabel/themes.sh`

Once these libraries are implemented, the tests will automatically activate and validate the functionality.

## Test Implementation Details

### Prerequisites Check

The test suite checks for the existence of white-label libraries before running:

```bash
WHITELABEL_LIBS_AVAILABLE=false
if [[ -f "$PROJECT_ROOT/src/lib/whitelabel/branding.sh" ]] && \
   [[ -f "$PROJECT_ROOT/src/lib/whitelabel/domains.sh" ]] && \
   [[ -f "$PROJECT_ROOT/src/lib/whitelabel/email-templates.sh" ]] && \
   [[ -f "$PROJECT_ROOT/src/lib/whitelabel/themes.sh" ]]; then
  WHITELABEL_LIBS_AVAILABLE=true
fi
```

### Helper Function

All tests use a helper function to check availability:

```bash
check_whitelabel_available() {
  if [[ "$WHITELABEL_LIBS_AVAILABLE" != "true" ]]; then
    skip "White-label libraries not yet implemented"
    return 1
  fi
  return 0
}
```

### Test Assets

The suite creates temporary test assets:

- **Test Logos**: SVG logo files for upload testing
- **Test Themes**: JSON theme configuration files
- **Test CSS**: Custom CSS stylesheets
- **Test Email Templates**: HTML email templates with variable injection

All test assets are cleaned up automatically after test execution.

## Expected Behavior

When libraries are implemented, tests will validate:

1. **Branding**
   - Brand creation and management
   - Color scheme customization
   - Logo upload and management
   - Custom CSS application
   - Font customization
   - Preview functionality
   - Import/export of configurations

2. **Custom Domains**
   - Domain addition and removal
   - DNS verification
   - SSL certificate provisioning
   - Auto-renewal configuration
   - Health monitoring
   - Wildcard and subdomain support
   - HTTPS enforcement

3. **Email Templates**
   - Template listing and editing
   - Variable injection
   - Preview rendering
   - Test email sending
   - Multi-language support
   - Subject and from address customization

4. **Themes**
   - Theme creation and editing
   - Dark/light mode support
   - CSS variable overrides
   - Preview functionality
   - Import/export
   - Responsive design support

5. **Multi-Tenant**
   - Tenant isolation
   - Per-tenant branding
   - Per-tenant domains
   - Per-tenant email templates
   - Branding inheritance
   - Subdomain routing

## Integration with nself CLI

Tests validate the `nself whitelabel` CLI commands:

```bash
# Branding
nself whitelabel branding create <name>
nself whitelabel branding set-colors --primary #hex --secondary #hex
nself whitelabel branding set-fonts --primary "Font" --secondary "Font"
nself whitelabel logo upload <path> [--type main|icon|email]

# Domains
nself whitelabel domain add <domain>
nself whitelabel domain verify <domain>
nself whitelabel domain ssl <domain> [--auto-renew]
nself whitelabel domain health <domain>
nself whitelabel domain remove <domain>

# Email
nself whitelabel email list
nself whitelabel email edit <template>
nself whitelabel email preview <template>
nself whitelabel email test <template> <email>
nself whitelabel email set-language <lang>

# Themes
nself whitelabel theme create <name>
nself whitelabel theme edit <name>
nself whitelabel theme activate <name>
nself whitelabel theme preview <name>
nself whitelabel theme export <name>
nself whitelabel theme import <path>

# Multi-tenant
nself whitelabel <command> --tenant <tenant-id>
```

## Success Criteria

All 55 tests should pass when:

1. White-label libraries are implemented
2. CLI commands are functional
3. Database tables exist for storing configurations
4. File storage is configured for logos/assets
5. Nginx/routing supports custom domains
6. Email system supports template customization
7. Multi-tenant isolation is properly configured

## Related Documentation

- Sprint 14: White-Label & Customization (60pts)
- nself CLI: `src/cli/whitelabel.sh`
- White-label libraries: `src/lib/whitelabel/`
- Integration test framework: `src/tests/test_framework.sh`

## Maintenance

When adding new white-label features:

1. Add corresponding test functions to this file
2. Follow the naming convention: `test_<category>_<feature>()`
3. Use the `check_whitelabel_available || return 0` pattern
4. Add clear descriptions with `describe "..."`
5. Update this README with the new test count
6. Ensure tests are idempotent and don't leave artifacts

## CI/CD Integration

These tests can be integrated into CI/CD pipelines:

```yaml
- name: Run White-Label Tests
  run: bash src/tests/integration/test-whitelabel.sh
```

Tests will skip gracefully if libraries aren't implemented, allowing for incremental development.

---

**Test Suite Version**: 1.0.0
**Created**: 2026-01-29
**Sprint**: 14 (White-Label & Customization - 60pts)
**nself Version**: 0.9.0
