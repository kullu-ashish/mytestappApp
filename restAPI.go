package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"

	fileRW "restAPI/jsonFileIO"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

var machines []*fileRW.Machine

func main() {
	flag.Parse()

	filerw := fileRW.Init{
		"folder.json",
	}

	F1 := fileRW.Folder{"C:\\Temp", "0 0 0 1 1 ? 1970"}
	F2 := fileRW.Folder{"C:\\Delta", "0 0 0 1 1 ? 1970"}

	F3 := fileRW.Folder{"C:\\Delta1", "0 0 0 1 1 ? 1970"}
	F4 := fileRW.Folder{"C:\\Delta2", "0 0 0 1 1 ? 1970"}

	M1 := fileRW.Machine{
		Name: "ZTSQL01",
		Folders: []fileRW.Folder{
			F1,
			F2}}

	M2 := fileRW.Machine{
		Name: "ZTSQL02",
		Folders: []fileRW.Folder{
			F3,
			F4}}

	mArray := []*fileRW.Machine{&M1, &M2, &M1}

	filerw.WriteFile(mArray)

	machines, _ = filerw.ReadFile()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("root."))
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	// RESTy routes for "articles" resource
	r.Route("/machines", func(r chi.Router) {
		r.With(paginate).Get("/", ListMachines)
		r.Post("/", CreateMachine)       // POST /articles
		r.Get("/search", SearchMachines) // GET /articles/search

		r.Route("/{name}", func(r chi.Router) {
			r.Use(MachineCtx)            // Load the *Article on the request context
			r.Get("/", GetMachine)       // GET /articles/123
			r.Put("/", UpdateMachine)    // PUT /articles/123
			r.Delete("/", DeleteMachine) // DELETE /articles/123
		})

	})

	http.ListenAndServe(":3333", r)
}

func ListMachines(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("ListMachines")

	for _, machine := range machines {
		fmt.Printf("%v\n", *machine)
	}

	if err := render.RenderList(w, r, NewMachineListResponse(machines)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

}

func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// just a stub.. some ideas are to look at URL query params for something like
		// the page number, or the limit, and send a query cursor down the chain
		next.ServeHTTP(w, r)
	})
}

func (rd *MachineResponse) Render(w http.ResponseWriter, r *http.Request) error {
	fmt.Printf("MachineResponse::Render ")
	rd.m.Render(w, r)
	// Pre-processing before a response is marshalled and sent across the wire
	return nil
}

func NewMachineListResponse(machines []*fileRW.Machine) []render.Renderer {
	list := []render.Renderer{}
	for _, machine := range machines {
		list = append(list, NewMachineResponse(machine))
	}
	for _, l := range list {
		fmt.Printf("%+v\n", l)
		fmt.Printf("%#v\n", l)
	}
	return list
}

func init() {
	render.Respond = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if err, ok := v.(error); ok {

			// We set a default error status response code if one hasn't been set.
			if _, ok := r.Context().Value(render.StatusCtxKey).(int); !ok {
				w.WriteHeader(400)
			}

			// We log the error
			fmt.Printf("Logging err: %s\n", err.Error())

			// We change the response to not reveal the actual error message,
			// instead we can transform the message something more friendly or mapped
			// to some code / language, etc.
			render.DefaultResponder(w, r, render.M{"status": "error"})
			return
		}

		render.DefaultResponder(w, r, v)
	}
}

type MachineResponse struct {
	m fileRW.Machine
}

func NewMachineResponse(machine *fileRW.Machine) *MachineResponse {
	resp := &MachineResponse{m: *machine}
	fmt.Printf("\n==> %v\n", resp)
	return resp
}

func MachineCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var machine *fileRW.Machine
		var err error

		if mName := chi.URLParam(r, "name"); mName != "" {
			machine, err = dbGetMachine(mName)
		} else {
			render.Render(w, r, ErrNotFound)
			return
		}
		if err != nil {
			render.Render(w, r, ErrNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), "machine", machine)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

//===
func SearchMachines(w http.ResponseWriter, r *http.Request) {
	render.RenderList(w, r, NewMachineListResponse(machines))
}

// CreateArticle persists the posted Article and returns it
// back to the client as an acknowledgement.
func CreateMachine(w http.ResponseWriter, r *http.Request) {
	data := &MachineRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	dbNewMachine(data.m)

	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewMachineResponse(data.m))
}

// GetArticle returns the specific Article. You'll notice it just
// fetches the Article right off the context, as its understood that
// if we made it this far, the Article must be on the context. In case
// its not due to a bug, then it will panic, and our Recoverer will save us.
func GetMachine(w http.ResponseWriter, r *http.Request) {
	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the ArticleCtx
	// middleware. The worst case, the recoverer middleware will save us.
	machine := r.Context().Value("machine").(*fileRW.Machine)

	if err := render.Render(w, r, NewMachineResponse(machine)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// UpdateArticle updates an existing Article in our persistent store.
func UpdateMachine(w http.ResponseWriter, r *http.Request) {
	machine := r.Context().Value("machine").(*fileRW.Machine)

	data := &MachineRequest{m: machine}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	dbUpdateMachine(data.m.Name, data.m)

	render.Render(w, r, NewMachineResponse(machine))
}

// DeleteArticle removes an existing Article from our persistent store.
func DeleteMachine(w http.ResponseWriter, r *http.Request) {
	var err error

	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the ArticleCtx
	// middleware. The worst case, the recoverer middleware will save us.
	machine := r.Context().Value("machine").(*fileRW.Machine)

	machine, err = dbRemoveMachine(machine.Name)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	render.Render(w, r, NewMachineResponse(machine))
}

type MachineRequest struct {
	m *fileRW.Machine
}

func (mr *MachineRequest) Bind(r *http.Request) error {
	// a.Article is nil if no Article fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if mr.m == nil {
		return errors.New("missing required Machine fields.")
	}
	return nil
}

func dbNewMachine(machine *fileRW.Machine) (string, error) {
	machines = append(machines, machine)
	return machine.Name, nil
}

func dbGetMachine(name string) (*fileRW.Machine, error) {
	for _, a := range machines {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, errors.New("machine not found.")
}

func dbUpdateMachine(name string, machine *fileRW.Machine) (*fileRW.Machine, error) {
	for i, a := range machines {
		if a.Name == name {
			machines[i] = machine
			return machine, nil
		}
	}
	return nil, errors.New("machine not found.")
}

func dbRemoveMachine(name string) (*fileRW.Machine, error) {
	for i, a := range machines {
		if a.Name == name {
			machines = append((machines)[:i], (machines)[i+1:]...)
			return a, nil
		}
	}
	return nil, errors.New("machine not found.")
}

//--
// Error response payloads & renderers
//--

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}
