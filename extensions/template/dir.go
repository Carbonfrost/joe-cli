package template

type dirGenerator struct {
	name     string
	contents []Generator
}

func Dir(name string, contents ...Generator) Generator {
	return &dirGenerator{name, contents}
}

func (d *dirGenerator) Generate(c *Context) error {
	c.PushDir(d.name)
	for _, g := range d.contents {
		err := c.Do(g)
		if err != nil {
			return err
		}
	}

	return c.PopDir()
}
