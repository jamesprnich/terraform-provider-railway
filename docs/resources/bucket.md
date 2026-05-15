---
page_title: "railway_bucket Resource - terraform-provider-railway"
subcategory: ""
description: |-
  Railway S3-compatible bucket.
---

# railway_bucket (Resource)

Railway bucket. S3-compatible object storage bucket attached to a project.

~> **No delete API:** Railway does not currently expose a `bucketDelete` mutation. Running `tofu destroy` removes the bucket from Terraform state, but the bucket itself **remains in Railway** until the project is deleted or the bucket is removed manually via the Railway dashboard. Once Railway adds a delete API, this provider will be updated to call it.

## Example Usage

```terraform
resource "railway_bucket" "data" {
  name       = "user-uploads"
  project_id = railway_project.main.id
}
```

To obtain the S3-compatible credentials (access key ID, secret access key, endpoint), use the Railway dashboard or the `bucketS3Credentials` API directly. Credentials are not exposed through this Terraform resource.

## Schema

### Required

- `name` (String) Name of the bucket.
- `project_id` (String) Identifier of the project the bucket belongs to.

### Read-Only

- `id` (String) Identifier of the bucket.

## Import

Import is supported using the following syntax:

```shell
tofu import railway_bucket.data <project_id>:<bucket_id>
```
