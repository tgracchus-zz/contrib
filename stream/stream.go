package stream

import "context"

func NewStream(ctx context.Context, src Source) Stream {
	out := make(chan *Object)
	errs := make(chan error)
	s := &DefaultStream{out, errs}
	go func() {
		err := src(ctx, s)
		if err != nil {
			errs <- err
			s.closeError()
		}
		s.closeStream()

	}()

	return s
}

type Source func(ctx context.Context, s Stream) error

func Map(ctx context.Context, s Stream, mapF MapFunc) Stream {
	out := make(chan *Object)
	newSource := &DefaultStream{out, s.errors()}

	go func() {
		defer newSource.closeStream()
		for {
			select {
			case object := <-s.objects():
				newObject, err := mapF(ctx, object)
				if err != nil {
					s.errors() <- err
					s.closeError()
				}
				newSource.out <- newObject
			case <-ctx.Done():
				return
			}
		}

	}()

	return newSource
}

type MapFunc func(ctx context.Context, object *Object) (*Object, error)

func Subscribe(ctx context.Context, s Stream) ([]*Object, error) {
	var objects []*Object
	var err error

	for {
		select {
		case object := <-s.objects():
			objects = append(objects, object)
		case <-ctx.Done():
			return objects, nil
		case erre := <-s.errors():
			return objects, erre
		}
	}

	return objects, err
}

type Object struct {
	Data       map[string]interface{}
	ObjectType string
}

type Stream interface {
	Push(object *Object)
	objects() chan *Object
	closeStream()

	errors() chan error
	closeError()
}

type DefaultStream struct {
	out  chan *Object
	errs chan error
}

func (src *DefaultStream) Push(object *Object) {
	src.out <- object
}

func (src *DefaultStream) objects() chan *Object {
	return src.out
}
func (src *DefaultStream) closeStream() {
	close(src.out)
}

func (src *DefaultStream) closeError() {
	close(src.errs)
}

func (src *DefaultStream) errors() chan error {
	return src.errs
}
