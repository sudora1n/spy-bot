// ЭТО ВСЕ - AI GENERATED
// Не приветствую такое, но нет желания с этим работать, а для вебаппа хочется нормально структурированные
// сообщения в bson, а не строкой в json, тут все обработчики, что нужны для корректного decode в telego.Message для v1.1.1

package custom_registry

import (
	// Добавлен для NewEncoder/NewDecoder
	"bytes"
	"errors"
	"reflect"

	"github.com/mymmrac/telego"
	"go.mongodb.org/mongo-driver/bson" // Используется для bson.Type* и нового API реестра/кодировщика
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	// bsontype больше не нужен напрямую, если используем bson.Type*
)

// PaidMediaCodec handles encoding/decoding of PaidMedia interface
type PaidMediaCodec struct{}

// EncodeValue encodes PaidMedia interface to BSON
func (pmc *PaidMediaCodec) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.IsNil() {
		return vw.WriteNull()
	}

	paidMedia, ok := val.Interface().(telego.PaidMedia)
	if !ok {
		return errors.New("value is not telego.PaidMedia")
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	// Write the media type
	mediaType := paidMedia.MediaType()
	// Исправлено: возврат к WriteDocumentElement + WriteString
	vwMediaType, err := dw.WriteDocumentElement("media_type")
	if err != nil {
		return err
	}
	if err := vwMediaType.WriteString(mediaType); err != nil {
		return err
	}

	// Write the actual data based on type
	vwData, err := dw.WriteDocumentElement("data")
	if err != nil {
		return err
	}

	var encodeErr error
	switch mediaType {
	case telego.PaidMediaTypePreview:
		preview := paidMedia.(*telego.PaidMediaPreview)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(preview))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(preview))
	case telego.PaidMediaTypePhoto:
		photo := paidMedia.(*telego.PaidMediaPhoto)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(photo))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(photo))
	case telego.PaidMediaTypeVideo:
		video := paidMedia.(*telego.PaidMediaVideo)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(video))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(video))
	default:
		encodeErr = errors.New("unknown PaidMedia type: " + mediaType)
	}

	if encodeErr != nil {
		return encodeErr
	}

	return dw.WriteDocumentEnd()
}

// DecodeValue decodes BSON to PaidMedia interface
func (pmc *PaidMediaCodec) DecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return errors.New("value is not settable")
	}
	// Исправлено: bsontype.Null -> bson.TypeNull
	if vr.Type() == bson.TypeNull {
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	}
	// Исправлено: bsontype.EmbeddedDocument -> bson.TypeEmbeddedDocument
	if vr.Type() != bson.TypeEmbeddedDocument {
		return errors.New("expected document for PaidMedia")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	var mediaType string
	var dataFound bool

	for {
		key, vrElementValue, err := dr.ReadElement()
		if errors.Is(err, bsonrw.ErrEOD) {
			break
		}
		if err != nil {
			return err
		}

		switch key {
		case "media_type":
			mediaType, err = vrElementValue.ReadString()
			if err != nil {
				return err
			}
		case "data":
			if mediaType == "" {
				return errors.New("media_type must come before data in BSON document")
			}

			var concreteMedia telego.PaidMedia
			var valueToDecode reflect.Value

			switch mediaType {
			case telego.PaidMediaTypePreview:
				var preview telego.PaidMediaPreview
				concreteMedia = &preview
				valueToDecode = reflect.ValueOf(&preview)
			case telego.PaidMediaTypePhoto:
				var photo telego.PaidMediaPhoto
				concreteMedia = &photo
				valueToDecode = reflect.ValueOf(&photo)
			case telego.PaidMediaTypeVideo:
				var video telego.PaidMediaVideo
				concreteMedia = &video
				valueToDecode = reflect.ValueOf(&video)
			default:
				if errSkip := vrElementValue.Skip(); errSkip != nil {
					return errSkip
				}
				return errors.New("unknown PaidMedia type during decode: " + mediaType)
			}

			if !valueToDecode.IsValid() {
				return errors.New("internal error: valueToDecode not set for media type: " + mediaType)
			}

			decoder, errL := dc.LookupDecoder(valueToDecode.Type())
			if errL != nil {
				return errL
			}
			if errD := decoder.DecodeValue(dc, vrElementValue, valueToDecode); errD != nil {
				return errD
			}

			val.Set(reflect.ValueOf(concreteMedia))
			dataFound = true
		default:
			if err := vrElementValue.Skip(); err != nil {
				return err
			}
		}
	}

	if !dataFound && mediaType != "" {
		return errors.New("data field not found or not decoded for identified PaidMedia type: " + mediaType)
	}
	return nil
}

// MessageOriginCodec handles encoding/decoding of MessageOrigin interface
type MessageOriginCodec struct{}

// EncodeValue encodes MessageOrigin interface to BSON
func (moc *MessageOriginCodec) EncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.IsNil() {
		return vw.WriteNull()
	}

	messageOrigin, ok := val.Interface().(telego.MessageOrigin)
	if !ok {
		return errors.New("value is not telego.MessageOrigin")
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	// Write the origin type
	originType := messageOrigin.OriginType()
	// Исправлено: возврат к WriteDocumentElement + WriteString
	vwOriginType, err := dw.WriteDocumentElement("origin_type")
	if err != nil {
		return err
	}
	if err := vwOriginType.WriteString(originType); err != nil {
		return err
	}

	// Write the actual data based on type
	vwData, err := dw.WriteDocumentElement("data")
	if err != nil {
		return err
	}

	var encodeErr error
	switch originType {
	case telego.OriginTypeUser:
		user := messageOrigin.(*telego.MessageOriginUser)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(user))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(user))
	case telego.OriginTypeHiddenUser:
		hiddenUser := messageOrigin.(*telego.MessageOriginHiddenUser)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(hiddenUser))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(hiddenUser))
	case telego.OriginTypeChat:
		chat := messageOrigin.(*telego.MessageOriginChat)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(chat))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(chat))
	case telego.OriginTypeChannel:
		channel := messageOrigin.(*telego.MessageOriginChannel)
		encoder, errL := ec.LookupEncoder(reflect.TypeOf(channel))
		if errL != nil {
			return errL
		}
		encodeErr = encoder.EncodeValue(ec, vwData, reflect.ValueOf(channel))
	default:
		encodeErr = errors.New("unknown MessageOrigin type: " + originType)
	}

	if encodeErr != nil {
		return encodeErr
	}

	return dw.WriteDocumentEnd()
}

// DecodeValue decodes BSON to MessageOrigin interface
func (moc *MessageOriginCodec) DecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() {
		return errors.New("value is not settable")
	}
	// Исправлено: bsontype.Null -> bson.TypeNull
	if vr.Type() == bson.TypeNull {
		val.Set(reflect.Zero(val.Type()))
		return vr.ReadNull()
	}
	// Исправлено: bsontype.EmbeddedDocument -> bson.TypeEmbeddedDocument
	if vr.Type() != bson.TypeEmbeddedDocument {
		return errors.New("expected document for MessageOrigin")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	var originType string
	var dataFound bool

	for {
		key, vrElementValue, err := dr.ReadElement()
		if errors.Is(err, bsonrw.ErrEOD) {
			break
		}
		if err != nil {
			return err
		}

		switch key {
		case "origin_type":
			originType, err = vrElementValue.ReadString()
			if err != nil {
				return err
			}
		case "data":
			if originType == "" {
				return errors.New("origin_type must come before data in BSON document")
			}

			var concreteOrigin telego.MessageOrigin
			var valueToDecode reflect.Value

			switch originType {
			case telego.OriginTypeUser:
				var user telego.MessageOriginUser
				concreteOrigin = &user
				valueToDecode = reflect.ValueOf(&user)
			case telego.OriginTypeHiddenUser:
				var hiddenUser telego.MessageOriginHiddenUser
				concreteOrigin = &hiddenUser
				valueToDecode = reflect.ValueOf(&hiddenUser)
			case telego.OriginTypeChat:
				var chat telego.MessageOriginChat
				concreteOrigin = &chat
				valueToDecode = reflect.ValueOf(&chat)
			case telego.OriginTypeChannel:
				var channel telego.MessageOriginChannel
				concreteOrigin = &channel
				valueToDecode = reflect.ValueOf(&channel)
			default:
				if errSkip := vrElementValue.Skip(); errSkip != nil {
					return errSkip
				}
				return errors.New("unknown MessageOrigin type during decode: " + originType)
			}

			if !valueToDecode.IsValid() {
				return errors.New("internal error: valueToDecode not set for origin type: " + originType)
			}

			decoder, errL := dc.LookupDecoder(valueToDecode.Type())
			if errL != nil {
				return errL
			}
			if errD := decoder.DecodeValue(dc, vrElementValue, valueToDecode); errD != nil {
				return errD
			}

			val.Set(reflect.ValueOf(concreteOrigin))
			dataFound = true
		default:
			if err := vrElementValue.Skip(); err != nil {
				return err
			}
		}
	}

	if !dataFound && originType != "" {
		return errors.New("data field not found or not decoded for identified MessageOrigin type: " + originType)
	}
	return nil
}

type CustomRegistry struct {
	Registry *bsoncodec.Registry
}

func CreateCustomRegistry() *CustomRegistry {
	// Исправлено: bsoncodec.NewRegistryBuilder -> bson.NewRegistry()
	// Удалена явная регистрация DefaultValueEncoders/Decoders и rb.Build()
	registry := bson.NewRegistry()

	paidMediaType := reflect.TypeOf((*telego.PaidMedia)(nil)).Elem()
	// Исправлено: rb.RegisterTypeEncoder -> registry.RegisterTypeEncoder
	registry.RegisterTypeEncoder(paidMediaType, &PaidMediaCodec{})
	registry.RegisterTypeDecoder(paidMediaType, &PaidMediaCodec{})

	messageOriginType := reflect.TypeOf((*telego.MessageOrigin)(nil)).Elem()
	registry.RegisterTypeEncoder(messageOriginType, &MessageOriginCodec{})
	registry.RegisterTypeDecoder(messageOriginType, &MessageOriginCodec{})

	customRegistry := CustomRegistry{
		Registry: registry,
	}
	return &customRegistry
}

// LoadMessage loads a telego.Message (or any struct containing these interfaces) from BSON using custom registry
func (c *CustomRegistry) LoadMessage(data bson.Raw, message *telego.Message) error {
	// Use bsonrw.NewBSONDocumentReader when you have the BSON data as a byte slice.
	// This function expects the raw BSON bytes of a single document.
	valueReader := bsonrw.NewBSONDocumentReader(data)
	decoder, err := bson.NewDecoder(valueReader)
	if err != nil {
		// This error is less likely if valueReader was created successfully from valid BSON.
		return errors.New("failed to create BSON decoder: " + err.Error())
	}
	decoder.SetRegistry(c.Registry)
	decoder.UseJSONStructTags()

	return decoder.Decode(message)
}

// Assume CreateCustomRegistry is defined elsewhere.
// func CreateCustomRegistry() *bson.Registry { /* ... */ }

// The SaveMessage function would remain as previously corrected:
func (c *CustomRegistry) SaveMessage(message *telego.Message) ([]byte, error) {
	var buf bytes.Buffer // Import "bytes"

	valueWriter, err := bsonrw.NewBSONValueWriter(&buf)
	if err != nil {
		return nil, errors.New("failed to create BSON value writer: " + err.Error())
	}

	encoder, err := bson.NewEncoder(valueWriter)
	if err != nil {
		return nil, errors.New("failed to create BSON encoder: " + err.Error())
	}
	encoder.SetRegistry(c.Registry)
	encoder.UseJSONStructTags()

	if err := encoder.Encode(message); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
