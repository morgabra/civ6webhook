module github.com/morgabra/civ6webhook

go 1.14

replace github.com/nlopes/slack => github.com/jirwin/slack v0.6.1-0.20200216211639-aba6477b6931

require (
	github.com/jirwin/quadlek v0.0.0-20200219063135-4447aaca6847
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli/v2 v2.2.0
	go.uber.org/zap v1.14.1
)
