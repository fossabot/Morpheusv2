// Copyright © 2018 MTRNord <info@nordgedanken.de>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"github.com/Nordgedanken/Morpheusv2/pkg/matrix"
	"github.com/Nordgedanken/Morpheusv2/pkg/matrix/users"
	"github.com/Nordgedanken/Morpheusv2/pkg/util"
	"github.com/matrix-org/gomatrix"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/uitools"
	"github.com/therecipe/qt/widgets"
	"log"
	"strings"
)

// LoginUI defines the data for the login ui
type LoginUI struct {
	widget       *widgets.QWidget
	window       *widgets.QMainWindow
	windowWidth  int
	windowHeight int

	localpart      string
	password       string
	server         string
	passwordInput  *widgets.QLineEdit
	localpartInput *widgets.QLineEdit

	helloMatrixResp helloMatrixResp
	serverDropdown  *widgets.QComboBox
}

// NewLoginUI gives you a MainUI struct with profiled data
func NewLoginUI(windowWidth, windowHeight int, window *widgets.QMainWindow) (loginUI *LoginUI) {
	loginUI = &LoginUI{
		windowWidth:  windowWidth,
		windowHeight: windowHeight,
		window:       window,
	}
	return
}

// GetWidget returns the QWidget of the LoginUI
func (l *LoginUI) GetWidget() (widget *widgets.QWidget) {
	return l.widget
}

// NewUI prepares the new UI
func (l *LoginUI) NewUI() error {
	l.widget = widgets.NewQWidget(nil, 0)

	var loader = uitools.NewQUiLoader(nil)
	var file = core.NewQFile2(":/qml/ui/login.ui")

	file.Open(core.QIODevice__ReadOnly)
	loginWidget := loader.Load(file, l.widget)
	file.Close()

	var layout = widgets.NewQHBoxLayout()
	l.window.SetLayout(layout)
	layout.InsertWidget(0, loginWidget, 0, core.Qt__AlignTop|core.Qt__AlignLeft)
	layout.SetSpacing(0)
	layout.SetContentsMargins(0, 0, 0, 0)

	l.widget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)
	loginWidget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)

	l.widget.ConnectResizeEvent(func(event *gui.QResizeEvent) {
		loginWidget.Resize(event.Size())
		event.Accept()
	})

	// Run Setup for all fields
	go l.setupDropdown()
	go l.setupLocalpartInput()
	go l.setupPasswordInput()
	go l.setupLoginButton()
	go l.setupRegisterButton()

	l.window.SetWindowTitle("Morpheus - Login")

	return nil
}

func (l *LoginUI) setupLocalpartInput() {
	// LocalpartInput
	l.localpartInput = widgets.NewQLineEditFromPointer(l.widget.FindChild("LocalpartInput", core.Qt__FindChildrenRecursively).Pointer())

	l.localpartInput.ConnectTextChanged(func(value string) {
		if l.localpartInput.StyleSheet() == redBorder {
			l.localpartInput.SetStyleSheet("")
		}
		l.localpart = value
	})
}

func (l *LoginUI) setupPasswordInput() {
	// PasswordInput
	l.passwordInput = widgets.NewQLineEditFromPointer(l.widget.FindChild("PasswordInput", core.Qt__FindChildrenRecursively).Pointer())

	l.passwordInput.ConnectTextChanged(func(value string) {
		if l.passwordInput.StyleSheet() == redBorder {
			l.passwordInput.SetStyleSheet("")
		}
		l.password = value
	})

}

func (l *LoginUI) setupLoginButton() (err error) {
	// loginButton
	loginButton := widgets.NewQPushButtonFromPointer(l.widget.FindChild("LoginButton", core.Qt__FindChildrenRecursively).Pointer())

	loginButton.ConnectClicked(func(_ bool) {
		if l.localpart != "" && l.password != "" {
			l.server = l.serverDropdown.CurrentText()
			err = l.login()
			if err != nil {
				return
			}
		} else {
			if l.localpart == "" {
				l.localpartInput.SetStyleSheet(redBorder)
			} else {
				l.passwordInput.SetStyleSheet(redBorder)
			}
		}
	})
	return
}

func (l *LoginUI) setupRegisterButton() (err error) {
	// registerButton
	registerButton := widgets.NewQPushButtonFromPointer(l.widget.FindChild("RegisterButton", core.Qt__FindChildrenRecursively).Pointer())

	registerButton.ConnectClicked(func(_ bool) {
		registerUIs := NewRegisterUI(l.windowWidth, l.windowHeight, l.window)
		SetNewWindow(registerUIs, l.window, l.windowWidth, l.windowHeight)
	})

	return
}

func (l *LoginUI) login() (err error) {
	var user matrix.User
	user, err = loginUser(l.localpart, l.password, l.server)
	if err != nil {
		return
	}

	util.User = user

	go func() {
		err = util.DB.SaveCurrentUser(user)
		if err != nil {
			log.Panicln(err)
			return
		}
	}()

	mainUIs := NewMainUI(l.windowWidth, l.windowHeight, l.window)
	SetNewWindow(mainUIs, l.window, l.windowWidth, l.windowHeight)

	return
}

//getClient returns a Client
func getClient(homeserverURL string) (client *gomatrix.Client, err error) {
	client, ClientErr := gomatrix.NewClient(homeserverURL, "", "")
	if ClientErr != nil {
		err = ClientErr
		return
	}

	return
}

//loginUser Creates a Session for the User
func loginUser(localpart, password, homeserverURL string) (matrix.User, error) {
	var cli *gomatrix.Client
	var cliErr error
	if strings.HasPrefix(homeserverURL, "https://") {
		cli, cliErr = getClient(homeserverURL)
	} else if strings.HasPrefix(homeserverURL, "http://") {
		cli, cliErr = getClient(homeserverURL)
	} else {
		cli, cliErr = getClient("https://" + homeserverURL)
	}
	if cliErr != nil {
		return nil, cliErr
	}

	localpart = strings.Replace(localpart, "@", "", -1)

	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:                     "m.login.password",
		User:                     localpart,
		Password:                 password,
		InitialDeviceDisplayName: "Morpheus 0.1.0-Alpha",
	})
	if err != nil {
		return nil, err
	}

	cli.SetCredentials(resp.UserID, resp.AccessToken)

	user := &users.User{}
	user.SetCli(cli)
	user.SetMXID(cli.UserID)

	return user, nil
}

func (l *LoginUI) setupDropdown() (err error) {
	// ServerDropdown
	l.serverDropdown = widgets.NewQComboBoxFromPointer(l.widget.FindChild("ServerChooserDropdown", core.Qt__FindChildrenRecursively).Pointer())

	var helloMatrixRespErr error
	l.helloMatrixResp, helloMatrixRespErr = getHelloMatrixList()
	if helloMatrixRespErr != nil {
		log.Println(helloMatrixRespErr)
		err = helloMatrixRespErr
		return
	}

	hostnames := convertHelloMatrixRespToNameSlice(l.helloMatrixResp)
	l.serverDropdown.AddItems(hostnames)
	return nil
}