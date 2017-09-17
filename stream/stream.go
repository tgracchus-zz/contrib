package stream

import "context"

func NewStream(ctx context.Context, src Source) *Stream {
	out := make(chan *Object)
	errs := make(chan error)
	s := &Stream{out, errs, ctx}
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

type Object struct {
	Data       map[string]interface{}
	ObjectType string
}

type Stream struct {
	out  chan *Object
	errs chan error
	ctx  context.Context
}

func (src *Stream) Push(object *Object) {
	src.out <- object
}

type Source func(ctx context.Context, s *Stream) error

func (s *Stream) Map(mapF MapFunc) *Stream {
	out := make(chan *Object)
	newStream := &Stream{out, s.errs, s.ctx}

	go func() {
		defer close(newStream.out)
		for {
			select {
			case object, ok := <-s.out:
				if ok {
					newObject, err := mapF(s.ctx, object)
					if err != nil {
						s.errs <- err
						defer close(s.errs)
					}
					newStream.out <- newObject
				} else {
					return
				}
			case <-s.ctx.Done():
				return
			}
		}

	}()

	return newStream
}

type MapFunc func(ctx context.Context, object *Object) (*Object, error)

func (s *Stream) Subscribe() ([]*Object, error) {
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
		case <-s.ctx.Done():
			return objects, nil
		case erre := <-s.errs:
			return objects, erre
		}
	}

	return objects, err
}
