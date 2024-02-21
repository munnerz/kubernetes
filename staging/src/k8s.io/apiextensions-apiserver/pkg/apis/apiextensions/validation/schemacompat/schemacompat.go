/*
Copyright 2021 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schemacompat

import (
	"errors"
	"fmt"
	"reflect"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// EnsureStructuralSchemaCompatibility compares a new structural schema to an existing one, to ensure that the existing
// schema is a sub-schema of the new schema. In other words that means that all the documents validated by the existing schema
// will also be validated by the new schema, so that the new schema can be considered backward-compatible with the existing schema.
// If it's not the case, errors are reported for each incompatible schema change.
//
// PLEASE NOTE that the implementation is incomplete (it's ongoing work), but still consistent:
// if some Json Schema elements are changed and the comparison of this type of element is not yet implemented,
// then an incompatible change error is triggered explaining that the comparison on this element type is not supported.
// So there should never be any case when a schema is considered backward-compatible while in fact it is not.
//
// If the narrowExisting argument is true, then the LCD (Lowest-Common-Denominator) between existing schema and the new schema
// is built (when possible), and returned if no incompatible change was detected (like type change, etc...).
// If the narrowExisting argument is false, the existing schema is untouched and no LCD schema is calculated.
//
// In either case, when no errors are reported, it is ensured that either the existing schema or the calculated LCD
// is a sub-schema of the new schema.
func EnsureStructuralSchemaCompatibility(fldPath *field.Path, existingInternal, newInternal *apiextensions.JSONSchemaProps, narrowExisting bool) field.ErrorList {
	newStrucural, err := schema.NewStructural(newInternal)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, err)}
	}

	existingStructural, err := schema.NewStructural(existingInternal)
	if err != nil {
		return field.ErrorList{field.InternalError(fldPath, err)}
	}

	lcdStructural := existingStructural.DeepCopy()
	if err := lcdForStructural(fldPath, existingStructural, newStrucural, lcdStructural, narrowExisting); err != nil {
		return err
	}

	return nil
}

func checkTypesAreTheSame(fldPath *field.Path, existing, new *schema.Structural) field.ErrorList {
	var allErrs field.ErrorList
	if new.Type != existing.Type {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("type"), new.Type, fmt.Sprintf("The type changed (was %q, now %q)", existing.Type, new.Type)))
	}
	return allErrs
}

func checkUnsupportedValidation(fldPath *field.Path, existing, new interface{}, validationName, typeName string) field.ErrorList {
	var allErrs field.ErrorList
	if !reflect.ValueOf(existing).IsZero() || !reflect.ValueOf(new).IsZero() {
		allErrs = append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("The %q JSON Schema construct is not supported by the Schema negotiation for type %q", validationName, typeName)))
	}
	return allErrs
}

func floatPointersEqual(p1, p2 *float64) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func intPointersEqual(p1, p2 *int64) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func stringPointersEqual(p1, p2 *string) bool {
	if p1 == nil && p2 == nil {
		return true
	}
	if p1 != nil && p2 != nil {
		return *p1 == *p2
	}
	return false
}

func checkUnsupportedValidationForNumerics(fldPath *field.Path, existing, new *schema.ValueValidation, typeName string) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Not, new.Not, "not", typeName)...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "allOf", typeName)...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AnyOf, new.AnyOf, "anyOf", typeName)...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.OneOf, new.OneOf, "oneOf", typeName)...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Enum, new.Enum, "enum", typeName)...)
	if !floatPointersEqual(new.Maximum, existing.Maximum) ||
		!floatPointersEqual(new.Minimum, existing.Minimum) ||
		new.ExclusiveMaximum != existing.ExclusiveMaximum ||
		new.ExclusiveMinimum != existing.ExclusiveMinimum {
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Maximum, new.Maximum, "maximum", typeName)...)
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Minimum, new.Minimum, "minimum", typeName)...)
	}
	if !floatPointersEqual(new.MultipleOf, existing.MultipleOf) {
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.MultipleOf, new.MultipleOf, "multipleOf", typeName)...)
	}
	return allErrs
}

func lcdForStructural(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	if lcd == nil && narrowExisting {
		return field.ErrorList{field.InternalError(fldPath, errors.New("lcd argument should be passed when narrowExisting is true"))}
	}
	if new == nil {
		return field.ErrorList{field.Invalid(fldPath, nil, "new schema doesn't allow anything")}
	}
	if was, now := existing.XPreserveUnknownFields, new.XPreserveUnknownFields; was != now {
		return field.ErrorList{field.Invalid(fldPath.Child("x-kubernetes-preserve-unknown-fields"), new.XPreserveUnknownFields, fmt.Sprintf("x-kubernetes-preserve-unknown-fields value changed (was %t, now %t)", was, now))}
	}

	switch existing.Type {
	case "number":
		return lcdForNumber(fldPath, existing, new, lcd, narrowExisting)
	case "integer":
		return lcdForInteger(fldPath, existing, new, lcd, narrowExisting)
	case "string":
		return lcdForString(fldPath, existing, new, lcd, narrowExisting)
	case "boolean":
		return lcdForBoolean(fldPath, existing, new, lcd, narrowExisting)
	case "array":
		return lcdForArray(fldPath, existing, new, lcd, narrowExisting)
	case "object":
		return lcdForObject(fldPath, existing, new, lcd, narrowExisting)
	case "":
		if existing.XIntOrString {
			return lcdForIntOrString(fldPath, existing, new, lcd, narrowExisting)
		} else if existing.XPreserveUnknownFields {
			return lcdForPreserveUnknownFields(fldPath, existing, new, lcd, narrowExisting)
		}
	}
	return field.ErrorList{field.Invalid(field.NewPath(fldPath.String(), "type"), existing.Type, "Invalid type")}
}

func lcdForIntegerValidation(fldPath *field.Path, existing, new *schema.ValueValidation, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	return checkUnsupportedValidationForNumerics(fldPath, existing, new, "integer")
}

func lcdForNumberValidation(fldPath *field.Path, existing, new *schema.ValueValidation, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	return checkUnsupportedValidationForNumerics(fldPath, existing, new, "numbers")
}

func lcdForNumber(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	if new.Type == "integer" {
		// new type is a subset of the existing type.
		if !narrowExisting {
			return checkTypesAreTheSame(fldPath, existing, new)
		}
		lcd.Type = new.Type
		return lcdForIntegerValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)
	}

	if err := checkTypesAreTheSame(fldPath, existing, new); err != nil {
		return err
	}

	return lcdForNumberValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)
}

func lcdForInteger(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	if new.Type == "number" {
		// new type is a superset of the existing type.
		// all is well type-wise
		// keep the existing type (integer) in the LCD
	} else {
		if err := checkTypesAreTheSame(fldPath, existing, new); err != nil {
			return err
		}
	}
	return lcdForIntegerValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)
}

func lcdForStringValidation(fldPath *field.Path, existing, new, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "allOf", "string")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "anytOf", "string")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "oneOf", "string")...)
	if !intPointersEqual(new.MaxLength, existing.MaxLength) ||
		!intPointersEqual(new.MinLength, existing.MinLength) {
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.MaxLength, new.MaxLength, "maxLength", "string")...)
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.MinLength, new.MinLength, "minLength", "string")...)
	}
	if new.Pattern != existing.Pattern {
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Pattern, new.Pattern, "pattern", "string")...)
	}
	toEnumSets := func(enum []schema.JSON) sets.String {
		enumSet := sets.NewString()
		for _, val := range enum {
			strVal, isString := val.Object.(string)
			if !isString {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("enum"), enum, "enum value should be a 'string' for Json type 'string'"))
				continue
			}
			enumSet.Insert(strVal)
		}
		return enumSet
	}
	existingEnumValues := toEnumSets(existing.Enum)
	newEnumValues := toEnumSets(new.Enum)
	if !newEnumValues.IsSuperset(existingEnumValues) {
		if !narrowExisting {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("enum"), newEnumValues.Difference(existingEnumValues).List(), "enum value has been changed in an incompatible way"))
		}
		lcd.Enum = nil
		lcdEnumValues := existingEnumValues.Intersection(newEnumValues).List()
		for _, val := range lcdEnumValues {
			lcd.Enum = append(lcd.Enum, schema.JSON{Object: val})
		}
	}

	if existing.Format != new.Format {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("format"), new.Format, "format value has been changed in an incompatible way"))
	}
	return allErrs
}

func lcdForString(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, lcdForStringValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)
	allErrs = append(allErrs, checkTypesAreTheSame(fldPath, existing, new)...)
	return allErrs
}

func lcdForBooleanValidation(fldPath *field.Path, existing, new, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "allOf", "boolean")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "anytOf", "boolean")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "oneOf", "boolean")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Enum, new.Enum, "enum", "boolean")...)
	return allErrs
}

func lcdForBoolean(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, lcdForBooleanValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)
	allErrs = append(allErrs, checkTypesAreTheSame(fldPath, existing, new)...)
	return allErrs
}

func lcdForArrayValidation(fldPath *field.Path, existing, new, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "allOf", "array")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "anytOf", "array")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "oneOf", "array")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Enum, new.Enum, "enum", "array")...)
	if !intPointersEqual(new.MaxItems, existing.MaxItems) ||
		!intPointersEqual(new.MinItems, existing.MinItems) {
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.MaxLength, new.MaxLength, "maxItems", "array")...)
		allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.MinLength, new.MinLength, "minItems", "array")...)
	}
	if !existing.UniqueItems && new.UniqueItems {
		if !narrowExisting {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("uniqueItems"), new.UniqueItems, "uniqueItems value has been changed in an incompatible way"))
		} else {
			lcd.UniqueItems = true
		}
	}
	return allErrs
}

func lcdForArray(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkTypesAreTheSame(fldPath, existing, new)...)
	allErrs = append(allErrs, lcdForArrayValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)
	allErrs = append(allErrs, lcdForStructural(fldPath.Child("Items"), existing.Items, new.Items, lcd.Items, narrowExisting)...)
	if !stringPointersEqual(existing.Extensions.XListType, new.Extensions.XListType) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("x-kubernetes-list-type"), new.Extensions.XListType, "x-kubernetes-list-type value has been changed in an incompatible way"))
	}
	if !sets.NewString(existing.Extensions.XListMapKeys...).Equal(sets.NewString(new.Extensions.XListMapKeys...)) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("x-kubernetes-list-map-keys"), new.Extensions.XListType, "x-kubernetes-list-map-keys value has been changed in an incompatible way"))
	}
	return allErrs
}

func lcdForObjectValidation(fldPath *field.Path, existing, new, lcd *schema.ValueValidation, narrowExisting bool) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "allOf", "object")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "anyOf", "object")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.AllOf, new.AllOf, "oneOf", "object")...)
	allErrs = append(allErrs, checkUnsupportedValidation(fldPath, existing.Enum, new.Enum, "enum", "object")...)
	return allErrs
}

func lcdForObject(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, checkTypesAreTheSame(fldPath, existing, new)...)

	if !stringPointersEqual(existing.Extensions.XMapType, new.Extensions.XMapType) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("x-kubernetes-map-type"), new.Extensions.XListType, "x-kubernetes-map-type value has been changed in an incompatible way"))
	}

	// Let's keep in mind that, in structural schemas, properties and additionalProperties are mutually exclusive,
	// which greatly simplifies the logic here.

	if len(existing.Properties) > 0 {
		if len(new.Properties) > 0 {
			existingProperties := sets.StringKeySet(existing.Properties)
			newProperties := sets.StringKeySet(new.Properties)
			lcdProperties := existingProperties
			if !newProperties.IsSuperset(existingProperties) {
				if !narrowExisting {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("properties"), existingProperties.Difference(newProperties).List(), "properties have been removed in an incompatible way"))
				}
				lcdProperties = existingProperties.Intersection(newProperties)
			}
			for _, key := range lcdProperties.List() {
				existingPropertySchema := existing.Properties[key]
				newPropertySchema := new.Properties[key]
				lcdPropertySchema := lcd.Properties[key]
				allErrs = append(allErrs, lcdForStructural(fldPath.Child("properties").Key(key), &existingPropertySchema, &newPropertySchema, &lcdPropertySchema, narrowExisting)...)
				lcd.Properties[key] = lcdPropertySchema
			}
			for _, removedProperty := range existingProperties.Difference(lcdProperties).UnsortedList() {
				delete(lcd.Properties, removedProperty)
			}
		} else if new.AdditionalProperties != nil && new.AdditionalProperties.Structural != nil {
			for _, key := range sets.StringKeySet(existing.Properties).List() {
				existingPropertySchema := existing.Properties[key]
				lcdPropertySchema := lcd.Properties[key]
				allErrs = append(allErrs, lcdForStructural(fldPath.Child("properties").Key(key), &existingPropertySchema, new.AdditionalProperties.Structural, &lcdPropertySchema, narrowExisting)...)
				lcd.Properties[key] = lcdPropertySchema
			}
		} else if new.AdditionalProperties != nil && new.AdditionalProperties.Bool {
			// that allows named properties only.
			// => Keep the existing schemas as the lcd.
		} else {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("properties"), sets.StringKeySet(existing.Properties).List(), "properties value has been completely cleared in an incompatible way"))
		}
	} else if existing.AdditionalProperties != nil {
		if existing.AdditionalProperties.Structural != nil {
			if new.AdditionalProperties.Structural != nil {
				allErrs = append(allErrs, lcdForStructural(fldPath.Child("additionalProperties"), existing.AdditionalProperties.Structural, new.AdditionalProperties.Structural, lcd.AdditionalProperties.Structural, narrowExisting)...)
			} else if existing.AdditionalProperties != nil && new.AdditionalProperties.Bool {
				// new schema allows any properties of any schema here => it is a superset of the existing schema
				// that allows any properties of a given schema.
				// => Keep the existing schemas as the lcd.
			} else {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("additionalProperties"), new.AdditionalProperties.Bool, "additionalProperties value has been changed in an incompatible way"))
			}
		} else if existing.AdditionalProperties.Bool {
			if !new.AdditionalProperties.Bool {
				if !narrowExisting {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("additionalProperties"), new.AdditionalProperties.Bool, "additionalProperties value has been changed in an incompatible way"))
				}
				lcd.AdditionalProperties.Bool = false
				lcd.AdditionalProperties.Structural = new.AdditionalProperties.Structural
			}
		}
	}

	allErrs = append(allErrs, lcdForObjectValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)

	return allErrs
}

func lcdForIntOrString(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, checkTypesAreTheSame(fldPath, existing, new)...)
	if !new.XIntOrString {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("x-kubernetes-int-or-string"), new.XIntOrString, "x-kubernetes-int-or-string value has been changed in an incompatible way"))
	}

	// We special-case IntOrString, since they are expected to have a fixed AnyOf value.
	// So we'll check the AnyOf separately and remove it from the further string-related or int-related validation
	// where anyOf is currently not supported.
	existingAnyOf := existing.ValueValidation.AnyOf
	newAnyOf := new.ValueValidation.AnyOf
	if !reflect.DeepEqual(existingAnyOf, newAnyOf) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("anyOf"), newAnyOf, "anyOf value has been changed in an incompatible way"))
	}
	existing.ValueValidation.AnyOf = nil
	new.ValueValidation.AnyOf = nil
	allErrs = append(allErrs, lcdForStringValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)
	allErrs = append(allErrs, lcdForIntegerValidation(fldPath, existing.ValueValidation, new.ValueValidation, lcd.ValueValidation, narrowExisting)...)
	existing.ValueValidation.AnyOf = existingAnyOf
	lcd.ValueValidation.AnyOf = existingAnyOf
	new.ValueValidation.AnyOf = newAnyOf

	return allErrs
}

func lcdForPreserveUnknownFields(fldPath *field.Path, existing, new *schema.Structural, lcd *schema.Structural, narrowExisting bool) field.ErrorList {
	return checkTypesAreTheSame(fldPath, existing, new)
}
