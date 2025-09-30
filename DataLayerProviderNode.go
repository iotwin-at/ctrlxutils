package ctrlxutils

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/datalayer"
	fbs "github.com/boschrexroth/ctrlx-datalayer-golang/v2/pkg/fbs/comm/datalayer"
)

// ----------------------------------------------------------------------------
// Types
// ----------------------------------------------------------------------------

type DataLayerProviderNode[T any] struct {
	ctx      context.Context
	provider *datalayer.Provider
	node     *datalayer.ProviderNode
	val      T
	config   DataLayerProviderNodeConfig
	subs     map[uint64]*datalayer.ProviderSubscription
	mux      sync.Mutex
}

type DataLayerProviderNodeConfig struct {
	Path    string // Path where this value will be available at datalayer
	Unit    string // Unit (e.g. kg)
	Desc    string // Description
	DescUrl string // Description URL
}

// ----------------------------------------------------------------------------
// Public
// ----------------------------------------------------------------------------

func (d *DataLayerProviderNode[T]) SetValue(val T) {
	d.val = val
	d.notify()
}

func (d *DataLayerProviderNode[T]) GetValue() T {
	return d.val
}

// ----------------------------------------------------------------------------
// Private
// ----------------------------------------------------------------------------

func (d *DataLayerProviderNode[T]) run() {
	d.provider.RegisterNode(d.config.Path, d.node)
	defer d.provider.UnregisterNode(d.config.Path)
	for {
		select {
		case e, ok := <-d.node.Channels().OnCreate:
			if !ok {
				return
			}
			d.onCreate(&e)
		case e, ok := <-d.node.Channels().OnRemove:
			if !ok {
				return
			}
			d.onRemove(&e)
		case e, ok := <-d.node.Channels().OnBrowse:
			if !ok {
				return
			}
			d.onBrowse(&e)
		case e, ok := <-d.node.Channels().OnRead:
			if !ok {
				return
			}
			d.onRead(&e)
		case e, ok := <-d.node.Channels().OnWrite:
			if !ok {
				return
			}
			d.onWrite(&e)
		case e, ok := <-d.node.Channels().OnMetadata:
			if !ok {
				return
			}
			d.onMetadata(&e)
		case e, ok := <-d.node.Channels().OnSubscribe:
			if !ok {
				return
			}
			d.onSubscribe(&e)
		case e, ok := <-d.node.Channels().OnUnsubscribe:
			if !ok {
				return
			}
			d.onUnsubscribe(&e)
		case <-d.node.Channels().Done:
			d.onDone()
			return
		case <-d.ctx.Done():
			d.onDone()
			return
		}
	}
}

// ----------------------------------------------------------------------------
// Eventhandler
// ----------------------------------------------------------------------------

func (d *DataLayerProviderNode[T]) onCreate(e *datalayer.ProviderNodeEventData) {
	e.Callback(datalayer.ResultOk, nil)
}

func (d *DataLayerProviderNode[T]) onRemove(e *datalayer.ProviderNodeEvent) {
	e.Callback(datalayer.ResultOk, nil)
}

func (d *DataLayerProviderNode[T]) onBrowse(e *datalayer.ProviderNodeEvent) {
	v := datalayer.NewVariant()
	defer datalayer.DeleteVariant(v)
	v.SetArrayString([]string{})
	e.Callback(datalayer.ResultOk, v)
}

func (d *DataLayerProviderNode[T]) onRead(e *datalayer.ProviderNodeEventData) {
	v, err := toVariant(d.val)
	if err != nil {
		e.Callback(datalayer.ResultTypeMismatch, nil)
		return
	}
	defer datalayer.DeleteVariant(v)
	e.Callback(datalayer.ResultOk, v)
}

func (d *DataLayerProviderNode[T]) onWrite(e *datalayer.ProviderNodeEventData) {
	v, err := ofVariant[T](e.Data)
	if err != nil {
		e.Callback(datalayer.ResultFailed, nil)
		return
	}
	d.SetValue(v)
	e.Callback(datalayer.ResultOk, e.Data)
}

func (d *DataLayerProviderNode[T]) onMetadata(e *datalayer.ProviderNodeEvent) {
	m := datalayer.NewMetaDataBuilder(datalayer.AllowedOperationRead|datalayer.AllowedOperationWrite, d.config.Desc, d.config.DescUrl)
	m.Unit(d.config.Unit).DisplayName(reflect.TypeOf(d.val).Name()).NodeClass(fbs.NodeClassVariable)
	m.AddReference(datalayer.ReferenceTypeRead, typeToVariantTypePath(d.val))
	m.AddReference(datalayer.ReferenceTypeWrite, typeToVariantTypePath(d.val))

	v := m.Build()
	defer datalayer.DeleteVariant(v)
	e.Callback(datalayer.ResultOk, v)
}

func (d *DataLayerProviderNode[T]) onSubscribe(e *datalayer.ProviderNodeSubscription) {
	d.mux.Lock()
	defer d.mux.Unlock()
	e.Subsciption.GetUniqueId()
	d.subs[e.Subsciption.GetUniqueId()] = e.Subsciption
}

func (d *DataLayerProviderNode[T]) onUnsubscribe(e *datalayer.ProviderNodeSubscription) {
	d.mux.Lock()
	defer d.mux.Unlock()
	delete(d.subs, e.Subsciption.GetUniqueId())
}

func (d *DataLayerProviderNode[T]) onDone() {
	datalayer.DeleteProviderNode(d.node)
}

func (d *DataLayerProviderNode[T]) notify() {
	for _, s := range d.subs {
		s.Publish(datalayer.ResultOk, []*datalayer.NotifyItemPublish{datalayer.NewNotifyItemPublish(d.config.Path)})
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// ToVariant converts a generic type T to a datalayer.Variant
func toVariant[T any](value T) (*datalayer.Variant, error) {
	variant := datalayer.NewVariant()

	// Handle special cases for structs
	if t, ok := any(value).(time.Time); ok {
		variant.SetTime(t)
		return variant, nil
	}

	// Use reflection to determine the actual type
	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Bool:
		variant.SetBool8(v.Bool())

	case reflect.Int8:
		variant.SetInt8(int8(v.Int()))

	case reflect.Int16:
		variant.SetInt16(int16(v.Int()))

	case reflect.Int32:
		variant.SetInt32(int32(v.Int()))

	case reflect.Int64:
		variant.SetInt64(v.Int())

	case reflect.Int:
		// Platform-dependent int - convert to int64 for safety
		variant.SetInt64(v.Int())

	case reflect.Uint8:
		variant.SetUint8(uint8(v.Uint()))

	case reflect.Uint16:
		variant.SetUint16(uint16(v.Uint()))

	case reflect.Uint32:
		variant.SetUint32(uint32(v.Uint()))

	case reflect.Uint64:
		variant.SetUint64(v.Uint())

	case reflect.Uint:
		// Platform-dependent uint - convert to uint64 for safety
		variant.SetUint64(v.Uint())

	case reflect.Float32:
		variant.SetFloat32(float32(v.Float()))

	case reflect.Float64:
		variant.SetFloat64(v.Float())

	case reflect.String:
		variant.SetString(v.String())
	case reflect.Slice, reflect.Array:
		return handleArrayType(variant, v)

	default:
		datalayer.DeleteVariant(variant)
		return nil, fmt.Errorf("unsupported type: %T", value)
	}

	return variant, nil
}

// handleArrayType processes slice and array types
func handleArrayType(variant *datalayer.Variant, v reflect.Value) (*datalayer.Variant, error) {
	if v.Len() == 0 {
		datalayer.DeleteVariant(variant)
		return nil, fmt.Errorf("empty array/slice not supported")
	}

	// Get the element type
	elemType := v.Type().Elem()

	switch elemType.Kind() {
	case reflect.Bool:
		arr := make([]bool, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Bool()
		}
		variant.SetArrayBool8(arr)

	case reflect.Int8:
		arr := make([]int8, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = int8(v.Index(i).Int())
		}
		variant.SetArrayInt8(arr)

	case reflect.Int16:
		arr := make([]int16, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = int16(v.Index(i).Int())
		}
		variant.SetArrayInt16(arr)

	case reflect.Int32:
		arr := make([]int32, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = int32(v.Index(i).Int())
		}
		variant.SetArrayInt32(arr)

	case reflect.Int64:
		arr := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Int()
		}
		variant.SetArrayInt64(arr)

	case reflect.Int:
		// Convert platform-dependent int to int64 array
		arr := make([]int64, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Int()
		}
		variant.SetArrayInt64(arr)

	case reflect.Uint8:
		arr := make([]uint8, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = uint8(v.Index(i).Uint())
		}
		variant.SetArrayUint8(arr)

	case reflect.Uint16:
		arr := make([]uint16, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = uint16(v.Index(i).Uint())
		}
		variant.SetArrayUint16(arr)

	case reflect.Uint32:
		arr := make([]uint32, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = uint32(v.Index(i).Uint())
		}
		variant.SetArrayUint32(arr)

	case reflect.Uint64:
		arr := make([]uint64, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Uint()
		}
		variant.SetArrayUint64(arr)

	case reflect.Uint:
		// Convert platform-dependent uint to uint64 array
		arr := make([]uint64, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Uint()
		}
		variant.SetArrayUint64(arr)

	case reflect.Float32:
		arr := make([]float32, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = float32(v.Index(i).Float())
		}
		variant.SetArrayFloat32(arr)

	case reflect.Float64:
		arr := make([]float64, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).Float()
		}
		variant.SetArrayFloat64(arr)

	case reflect.String:
		arr := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = v.Index(i).String()
		}
		variant.SetArrayString(arr)

	default:
		datalayer.DeleteVariant(variant)
		return nil, fmt.Errorf("unsupported array element type: %v", elemType.Kind())
	}
	return variant, nil
}

func typeToVariantTypePath[T any](val T) string {
	t := reflect.TypeFor[T]()

	// Handle special cases for structs
	if t == reflect.TypeFor[time.Time]() {
		return "types/datalayer/timestamp"
	}

	// Handle default types
	switch t.Kind() {
	case reflect.Bool:
		return "types/datalayer/bool8"

	case reflect.Int8:
		return "types/datalayer/int8"

	case reflect.Int16:
		return "types/datalayer/int16"

	case reflect.Int32:
		return "types/datalayer/int32"

	case reflect.Int64, reflect.Int:
		return "types/datalayer/int64"

	case reflect.Uint8:
		return "types/datalayer/uint8"

	case reflect.Uint16:
		return "types/datalayer/uint16"

	case reflect.Uint32:
		return "types/datalayer/uint32"

	case reflect.Uint64, reflect.Uint:
		return "types/datalayer/uint64"

	case reflect.Float32:
		return "types/datalayer/float32"

	case reflect.Float64:
		return "types/datalayer/float64"

	case reflect.String:
		return "types/datalayer/string"

	case reflect.Slice, reflect.Array:
		return getTypePathOfArray(t.Elem())

	default:
		return "unsupported type"
	}
}

func getTypePathOfArray(t reflect.Type) string {
	// Handle special cases for structs
	if t == reflect.TypeFor[time.Time]() {
		return "types/datalayer/array-of-timestamp"
	}

	switch t.Kind() {
	case reflect.Bool:
		return "types/datalayer/array-of-bool8"

	case reflect.Int8:
		return "types/datalayer/array-of-int8"

	case reflect.Int16:
		return "types/datalayer/array-of-int16"

	case reflect.Int32:
		return "types/datalayer/array-of-int32"

	case reflect.Int64, reflect.Int:
		return "types/datalayer/array-of-int64"

	case reflect.Uint8:
		return "types/datalayer/array-of-uint8"

	case reflect.Uint16:
		return "types/datalayer/array-of-uint16"

	case reflect.Uint32:
		return "types/datalayer/array-of-uint32"

	case reflect.Uint64, reflect.Uint:
		return "types/datalayer/array-of-uint64"

	case reflect.Float32:
		return "types/datalayer/array-of-float32"

	case reflect.Float64:
		return "types/datalayer/array-of-float64"

	case reflect.String:
		return "types/datalayer/array-of-string"

	case reflect.Slice, reflect.Array:
		return getTypePathOfArray(t.Elem())

	default:
		return "unsupported type"
	}
}

func ofVariant[T any](v *datalayer.Variant) (T, error) {
	var result T
	targetType := reflect.TypeFor[T]()
	resultValue := reflect.ValueOf(&result).Elem()

	switch targetType.Kind() {
	case reflect.Bool:
		val := v.GetBool8()
		resultValue.SetBool(val)
		return result, nil

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		val := v.GetInt64()
		if resultValue.OverflowInt(val) {
			return result, fmt.Errorf("value %d overflows %v", val, targetType)
		}
		resultValue.SetInt(val)
		return result, nil

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		val := v.GetUint64()
		if resultValue.OverflowUint(val) {
			return result, fmt.Errorf("value %d overflows %v", val, targetType)
		}
		resultValue.SetUint(val)
		return result, nil

	case reflect.Float32, reflect.Float64:
		val := v.GetFloat64()
		resultValue.SetFloat(float64(val))
		return result, nil

	case reflect.String:
		val := v.GetString()
		resultValue.SetString(val)
		return result, nil

	case reflect.Slice:
		return variantToSlice[T](v, targetType, resultValue)

	case reflect.Array:
		return variantToArray[T](v, targetType, resultValue)

	default:
		return result, fmt.Errorf("unsupported type: %v", targetType)
	}
}

func variantToSlice[T any](v *datalayer.Variant, targetType reflect.Type, resultValue reflect.Value) (T, error) {
	var result T
	elementType := targetType.Elem()

	// Get the appropriate slice based on element type
	var sourceSlice interface{}

	switch elementType.Kind() {
	case reflect.Bool:
		sourceSlice = v.GetArrayBool8()
	case reflect.Int8:
		sourceSlice = v.GetArrayInt8()
	case reflect.Uint8:
		sourceSlice = v.GetArrayUint8()
	case reflect.Int16:
		sourceSlice = v.GetArrayInt16()
	case reflect.Uint16:
		sourceSlice = v.GetArrayUint16()
	case reflect.Int32:
		sourceSlice = v.GetArrayInt32()
	case reflect.Uint32:
		sourceSlice = v.GetArrayUint32()
	case reflect.Float32:
		sourceSlice = v.GetArrayFloat32()
	case reflect.Float64:
		sourceSlice = v.GetArrayFloat64()
	case reflect.String:
		sourceSlice = v.GetArrayString()
	default:
		return result, fmt.Errorf("unsupported slice element type: %v", elementType)
	}

	// Convert to reflect.Value and set
	sourceValue := reflect.ValueOf(sourceSlice)

	// Type assertion to ensure compatibility
	if sourceValue.Type().AssignableTo(targetType) {
		resultValue.Set(sourceValue)
		return result, nil
	}

	return result, fmt.Errorf("cannot assign %v to %v", sourceValue.Type(), targetType)
}

func variantToArray[T any](v *datalayer.Variant, targetType reflect.Type, resultValue reflect.Value) (T, error) {
	var result T
	elementType := targetType.Elem()
	arrayLen := targetType.Len()

	// Get the source slice (arrays come as slices from your getters)
	var sourceSlice interface{}

	switch elementType.Kind() {
	case reflect.Bool:
		sourceSlice = v.GetArrayBool8()
	case reflect.Int8:
		sourceSlice = v.GetArrayInt8()
	case reflect.Uint8:
		sourceSlice = v.GetArrayUint8()
	case reflect.Int16:
		sourceSlice = v.GetArrayInt16()
	case reflect.Uint16:
		sourceSlice = v.GetArrayUint16()
	case reflect.Int32:
		sourceSlice = v.GetArrayInt32()
	case reflect.Uint32:
		sourceSlice = v.GetArrayUint32()
	case reflect.Float32:
		sourceSlice = v.GetArrayFloat32()
	case reflect.Float64:
		sourceSlice = v.GetArrayFloat64()
	case reflect.String:
		sourceSlice = v.GetArrayString()
	default:
		return result, fmt.Errorf("unsupported array element type: %v", elementType)
	}

	sourceValue := reflect.ValueOf(sourceSlice)

	// Check length compatibility
	if sourceValue.Len() != arrayLen {
		return result, fmt.Errorf("array length mismatch: expected %d, got %d", arrayLen, sourceValue.Len())
	}

	// Copy elements from slice to array
	for i := range arrayLen {
		resultValue.Index(i).Set(sourceValue.Index(i))
	}

	return result, nil
}

// ----------------------------------------------------------------------------
// Constructors
// ----------------------------------------------------------------------------

func NewDataLayerProviderNode[T any](ctx context.Context, provider *datalayer.Provider, config DataLayerProviderNodeConfig) *DataLayerProviderNode[T] {
	dln := &DataLayerProviderNode[T]{
		ctx:      ctx,
		provider: provider,
		node:     datalayer.NewProviderNode(),
		config:   config,
		val:      *new(T),
		subs:     map[uint64]*datalayer.ProviderSubscription{},
		mux:      sync.Mutex{},
	}
	go dln.run()
	return dln
}
