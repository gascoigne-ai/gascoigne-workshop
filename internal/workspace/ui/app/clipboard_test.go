package app

type fakeClip struct {
	text string
	err  error
}

func (f *fakeClip) Read() (string, error) { return f.text, f.err }
func (f *fakeClip) Write(string string) error {
	if f.err != nil {
		return f.err
	}
	f.text = string
	return nil
}
