<!-- 
    This doc is a placeholder for future documentation pages/markdown files with the intention to surface the assumptions the generator makes (currently in RFC). CLI documentation + generator config syntax should be documented elsewhere
-->

<!--
    TODO: Will need to update this document's wording if the tool changes from generating IR to generating Go code w/ IR library
    TODO: Need to update links to generator config file syntax when available
-->
# OpenAPI to Framework IR (Intermediate Representation) Generator Design

## Overview

The OpenAPI to Framework IR Generator (referred to in this documentation as **generator**) provides mapping between [OpenAPI Specification](https://www.openapis.org/) version 3.0 and 3.1, to Terraform Plugin Framework IR. This mapping currently includes resources, data sources, and provider schema information, all of which are identified with a [generator config file](./README.md).

As the OpenAPI specification (OAS) is designed to describe HTTP APIs in general, it doesn't have full parity with the Terraform Plugin Framework schema or code patterns. There are pieces of logic in the generator that make assumptions on what portions of the OAS to use when mapping to Framework IR, this design document intends to describe those assumptions in detail.

Users of the generator can adjust their OAS to match these assumptions, or suggest changes/customization via the [generator config file](./README.md).

## Determining the OAS Schema to map from CRUD operations

### Resources
The [generator config file](./README.md) defines the CRUD (Create, Read, Update, Delete) operations in an OAS. In those operations, the generator will search `Create` and `Read` operations for schemas to map to Framework IR. Multiple schemas will be [deep merged](#deep-merge-of-schemas-resources) and the final result will be the Resource schema represented in Framework IR.

#### OAS Schema order (resources)
- `Create` operation [requestBody](https://spec.openapis.org/oas/v3.1.0#requestBodyObject)
    - `requestBody` is the only schema **required** for resources, if not present will log a warning and skip the resource without mapping.
    - Will attempt to use `application/json` first, then will grab the first content-type if not found (alphabetical sort)
- `Create` operation [response](https://spec.openapis.org/oas/v3.1.0#responsesObject)
    - Will attempt to use `200` or `201` first, then will grab the first 2xx response code if not found (lexicographic sort)
    - Will attempt to use `application/json` first, then will grab the first content-type if not found (alphabetical sort)
- `Read` operation [response](https://spec.openapis.org/oas/v3.1.0#responsesObject)
    - Will attempt to use `200` or `201` first, then will grab the first 2xx response code if not found (lexicographic sort)
    - Will attempt to use `application/json` first, then will grab the first content-type if not found (alphabetical sort)
- `Read` operation [parameters](https://spec.openapis.org/oas/v3.1.0#parameterObject)
    - The generator will [deep merge](#deep-merge-of-schemas-resources) the parameters defined belong at the root of the schema.

#### Deep merge of schemas (resources)
All schemas found will be deep merged together, with the `requestBody` schema from the `Create` operation being the `main schema` that the others will be merged on top. The deep merge has the following characteristics:

- Only attribute name is compared, if the attribute doesn't already exist in the main schema, it will be added. Any mismatched types of the same name will not raise an error and priority will favor the `main schema`.
- Names are strictly compared, so `id` and `user_id` would be two separate attributes in a schema.
- Arrays and Objects will have their child properties merged, so `example_object.string_field` and `example_object.bool_field` will be merged into the same `SingleNestedAttribute` schema.

### Data Sources
<!-- TODO: Fill this out once data source implementation is complete -->
TBD

## Mapping OAS Schema to Plugin Framework Types

### OAS to Plugin Framework Attribute Types

For a given [OAS type](https://spec.openapis.org/oas/v3.1.0#data-types) and format combination, the following rules will be applied for mapping to Framework  attribute types. Not all Framework types are represented natively with OAS, those types are noted below in [Unsupported Attribute Types](#unsupported-attribute-types).

| Type (OAS) | Format (OAS)        | Items type (OAS `array`) | Plugin Framework Attribute Type                                                             |
|------------|---------------------|--------------------------|---------------------------------------------------------------------------------------------|
| `integer`  | -                   | -                        | `Int64Attribute`                                                                            |
| `number`   | `double` or `float` | -                        | `Float64Attribute`                                                                          |
| `number`   | (all others)        | -                        | `NumberAttribute`                                                                           |
| `string`   | -                   | -                        | `StringAttribute`                                                                           |
| `boolean`  | -                   | -                        | `BoolAttribute`                                                                             |
| `array`    | -                   | `object`                 | `ListNestedAttribute`                                                                       |
| `array`    | -                   | (all others)             | `ListAttribute` (nests with [element types](#oas-to-plugin-framework-element-types)) |
| `object`   | -                   | -                        | `SingleNestedAttribute`                                                                     |

#### Unsupported Attribute Types
- `ListNestedBlock`, `SetNestedBlock`, and `SingleNestedBlock`
    - While the Plugin Framework supports blocks, the Plugin Framework team encourages provider developers to prefer `ListNestedAttribute`, `SetNestedAttribute`, and `SingleNestedAttribute` for new provider development.
- `ObjectAttribute`
    - The generator will default to `SingleNestedAttribute` for object types to provide the additional schema information.
- `SetNestedAttribute`, `SetAttribute`, `MapNestedAttribute`, and `MapAttribute`
    - Mapping for these types is currently not supported, but will be considered in future versions.

### OAS to Plugin Framework Element Types

For attributes that don’t have additional schema information (currently only `ListAttribute`), the following rules will be applied for mapping from OAS type and format combinations, into Framework element types.

| Type (OAS) | Format (OAS)        | Items type (OAS `array`) | Plugin Framework Element Type   |
|------------|---------------------|--------------------------|---------------------------------|
| `integer`  | -                   | -                        | `Int64Type`                     |
| `number`   | `double` or `float` | -                        | `Float64Type`                   |
| `number`   | (all others)        | -                        | `NumberType`                    |
| `string`   | -                   | -                        | `StringType`                    |
| `boolean`  | -                   | -                        | `BoolType`                      |
| `array`    | -                   | (all)                    | `ListType`                      |
| `object`   | -                   | -                        | `ObjectType`                    |

### Required, Computed, and Optional

#### Resources
For resources, all fields marked in the OAS schema as [required](https://json-schema.org/understanding-json-schema/reference/object.html#required-properties) will be mapped as a [Required](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#required) attribute.

If not required, then the field will be mapped as [Computed](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#computed) and [Optional](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#optional).

#### Data Sources
<!-- TODO: Fill this out once data source implementation is complete -->
TBD

### Other field mapping

| Field (OAS)                                                                   | Field (Plugin Framework Schema)                                                                                                           |
|-------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------|
| [description](https://spec.openapis.org/oas/latest.html#rich-text-formatting) | [(Attribute).MarkdownDescription](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#markdowndescription-1) |
| [format (password)](https://spec.openapis.org/oas/latest.html#data-types)     | [(StringAttribute).Sensitive](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas#sensitive)                 |


## Multi-type Support

Generally, [multi-types](https://cswr.github.io/JsonSchema/spec/multiple_types/) are not supported by the generator as the Terraform Plugin Framework does not support multi-types. There is one specific scenario that is supported by the generator and that is any type that is combined with the `null` type, as any Plugin Framework attribute can hold a [null](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#null) type.

### Nullable Multi-type support
> **Note:** with nullable multi-types, the `description` will be populated from the root-level schema, as shown below. 

In an OAS schema, the following keywords defining nullable multi-types are supported (nullable types will follow the same mapping rules [defined above](#oas-to-plugin-framework-attribute-types) for the type that is not the `null` type):

#### `type` keyword array
```jsonc
// Maps to StringAttribute
{
  "nullable_string_example": {
    "description": "this is the description that's used!",
    "type": [
      "string",
      "null"
    ]
  }
}

// Maps to Int64Attribute
{
  "nullable_integer_example": {
    "description": "this is the description that's used!",
    "type": [
      "null",
      "integer"
    ]
  }
}
```

#### `anyOf` and `oneOf` keywords
```jsonc
// Maps to SingleNestedAttribute
{
  "nullable_object_one": {
    "description": "this is the description that's used!",
    "anyOf": [
      {
        "type": "null"
      },
      {
        "$ref": "#/components/schemas/example_object_one"
      }
    ]
  }
}

// Maps to SingleNestedAttribute
{
  "nullable_object_two": {
    "description": "this is the description that's used!",
    "oneOf": [
      {
        "$ref": "#/components/schemas/example_object_two"
      },
      {
        "type": "null"
      }
    ]
  }
}
```