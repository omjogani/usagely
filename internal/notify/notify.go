package notify

import "github.com/godbus/dbus/v5"

const timeoutMillis = 30000

func Send(summary, body string) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	object := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	return object.Call("org.freedesktop.Notifications.Notify", 0,
		"Usagely", uint32(0), "dialog-warning",
		summary, body,
		[]string{}, map[string]dbus.Variant{}, int32(timeoutMillis),
	).Err
}
