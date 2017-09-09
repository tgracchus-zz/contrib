package stream

import "context"

func NewStream(ctx context.Context, src Source) *Stream {
	out := make(chan *Object)
	errs := make(chan error)
	s := &Stream{out, errs}
	go func() {
		err := src(ctx, s)
		if err != nil {
			errs <- err
			defer close(s.errs)
		}
		defer close(s.out)

	}()

	return s
}

type Source func(ctx context.Context, s *Stream) error

func Map(ctx context.Context, s Stream, mapF MapFunc) *Stream {
	out := make(chan *Object)
	newStream := &Stream{out, s.errs}

	go func() {
		defer close(newStream.out)
		for {
			select {
			case object := <-s.out:
				newObject, err := mapF(ctx, object)
				if err != nil {
					s.errs <- err
					defer close(s.errs)
				}
				newStream.out <- newObject
			case <-ctx.Done():
				return
			}
		}

	}()

	return newStream
}

type MapFunc func(ctx context.Context, object *Object) (*Object, error)

func Subscribe(ctx context.Context, s *Stream) ([]*Object, error) {
	var objects []*Object
	var err error

	for {
		select {
		case object, ok := <-s.out:
			if ok {
				objects = append(objects, object)
			} else {
				return objects, nil
			}
		case <-ctx.Done():
			return objects, nil
		case erre := <-s.errs:
			return objects, erre
		}
	}

	return objects, err
}

type Object struct {
	Data       map[string]interface{}
	ObjectType string
}

type Stream struct {
	out  chan *Object
	errs chan error
}

func (src *Stream) Push(object *Object) {
	src.out <- object
}
