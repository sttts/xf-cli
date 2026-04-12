package cmds

func (cmd *LoginCmd) Run(app *App) error {
	_, session, err := app.login()
	if err != nil {
		return err
	}

	return app.printSession(session)
}

type LoginCmd struct{}
