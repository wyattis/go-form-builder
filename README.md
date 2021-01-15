#  multiform
A Reader based interface to easily stream `multipart/form-data` encoded data.

## Install
`go get github.com/wyattis/multiform`

## Usage
The builder interface is analogous to the [multipart.Writer] interface in the standard lib, but much simpler to use. Most of the code to encode the form is copied verbatim from the standard package.

### Simple example
```
file, _ := os.Open("large-file.zip")

form := multiform.NewBuilder()
form.AddField("key", "value")
form.AddFormFile("file", file)
form.Done()

res, err := http.Post("example.com", form.FormDataContentType(), form)
```

### Custom boundary
Custom boundary can be called anytime before the form is read.
```
form := multiform.NewBuilder()
form.SetBoundary("custom-boundary")
form.AddFormFile("file", file)
form.Done()
res, err := http.Post("example.com", form.FormDataContentType(), form)
```

[multipart.Writer]: https://golang.org/pkg/mime/multipart/#Writer