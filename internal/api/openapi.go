package api

// openAPISpec descreve exatamente as rotas registradas em server.go. Rota nova entra aqui
// junto; rota que ainda nao existe nao entra (antes a spec listava 18 rotas inexistentes).
func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Arcanum API",
			"description": "Open-source, self-hosted WhatsApp gateway written in Go: multi-session, built-in anti-ban, signed webhooks.\n\nEvery route except `/health` and `/docs` requires the API key (`X-Api-Key` header or `Authorization: Bearer`). Full guide: https://arcanum.rafaeldourado.tech/en/",
			"version":     Version,
			"contact": map[string]string{
				"name": "Arcanum",
				"url":  "https://github.com/rafaeldourado9/arcanum",
			},
			"license": map[string]string{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]string{
			{"url": "/", "description": "This gateway"},
		},
		"security": []map[string][]string{{"ApiKeyHeader": {}}, {"BearerAuth": {}}},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"ApiKeyHeader": map[string]string{"type": "apiKey", "in": "header", "name": "X-Api-Key"},
				"BearerAuth":   map[string]string{"type": "http", "scheme": "bearer"},
			},
		},
		"tags": []map[string]string{
			{"name": "Instance", "description": "Create, connect (QR code or pairing code), inspect and remove WhatsApp sessions"},
			{"name": "Message", "description": "Send text, media, location, contacts, reactions and polls"},
			{"name": "Chat", "description": "Read receipts, number lookup and profile"},
			{"name": "Events", "description": "Webhook configuration per instance"},
			{"name": "Settings", "description": "Per-instance behaviour"},
			{"name": "System", "description": "Health, metrics, logs and this documentation"},
			{"name": "Auth", "description": "Dashboard login (browser session). API clients use the API key instead."},
		},
		"paths": buildPaths(),
	}
}

func buildPaths() map[string]any {
	return map[string]any{
		// ── System ──
		"/health":          public(pathGet("System", "Health check", "Gateway status, version and session counts by status. Public: no key needed, no session names or numbers.")),
		"/metrics":         pathGet("System", "Prometheus metrics", "Counters in the Prometheus text format: sessions by status, messages received/sent, webhook deliveries, HTTP responses by code."),
		"/api/stats":       pathGet("System", "Dashboard stats", "Version, uptime, sessions by status and message/webhook counters as JSON."),
		"/api/logs":        pathGet("System", "Recent logs", "Last log lines (`?limit=200&level=warn`). No message content or phone numbers; secrets are redacted."),
		"/api/logs/stream": pathGet("System", "Live logs", "Server-Sent Events stream of new log lines (`text/event-stream`)."),

		// ── Dashboard login ──
		"/api/auth/login": public(map[string]any{"post": opBody("Auth", "Dashboard login", "Exchange the dashboard password for an HttpOnly session cookie. 10 failures per IP in 15 minutes lock the login (429).", bodyObj(map[string]any{
			"password": strProp("GATEWAY_DASHBOARD_PASSWORD, or the API key when no password is set", ""),
		}))}),
		"/api/auth/logout": public(map[string]any{"post": op("Auth", "Dashboard logout", "End the dashboard session.")}),
		"/api/auth/me":     public(pathGet("Auth", "Session status", "Whether the browser has a valid dashboard session.")),

		// ── Instance ──
		"/api/instance/create": pathPost("Instance", "Create instance", "Create a session. `instanceName`: 1-64 letters, digits, '-' or '_'. Returns 201; 409 if the name exists; 400 if the name or webhook URL is invalid.", bodyObj(map[string]any{
			"instanceName":  strProp("Session name (1-64 of A-Z a-z 0-9 - _)", "sales"),
			"webhook":       strProp("Optional webhook URL for this session (http/https)", "https://example.com/webhooks/whatsapp"),
			"webhookEvents": arrProp("Events to subscribe", []string{"messages", "status"}),
		})),
		"/api/instance/connect/{instance}":         pathGetQR("Instance", "Connect instance", "Start the connection and return the QR code (waits briefly for it). Calling it again while waiting returns the same QR; it never opens a second client."),
		"/api/instance/connectionState/{instance}": pathGetQR("Instance", "Connection state", "Current status (disconnected, connecting, qr, connected) and the QR code while waiting for the scan."),
		"/api/instance/qr/{instance}":              pathGetQR("Instance", "Get QR code", "Current QR code of a pending connection. 404 when there is none."),
		"/api/instance/pair/{instance}": pathPost("Instance", "Pair by phone number", "Alternative to the QR code: returns an 8-character code to type in WhatsApp > Linked devices > Link with phone number.", bodyObj(map[string]any{
			"phone": strProp("Phone number with country code, digits only", "5511999999999"),
		})),
		"/api/instance/logout/{instance}":  pathDelParam("Instance", "Disconnect instance", "Disconnect the session. It stays paired and can reconnect with /restart.", "instance"),
		"/api/instance/delete/{instance}":  pathDelParam("Instance", "Delete instance", "Close the session and remove it from the gateway.", "instance"),
		"/api/instance/restart/{instance}": pathPostParam("Instance", "Restart instance", "Reconnect the session.", "instance"),
		"/api/instance/setPresence/{instance}": pathPost("Instance", "Set presence", "Set the account presence.", bodyObj(map[string]any{
			"presence": strProp("available or unavailable", "available"),
		})),
		"/api/instance/fetchInstances": pathGet("Instance", "List instances", "All sessions with status, settings and webhook configuration."),

		// ── Message ──
		"/api/message/sendText/{instance}": pathPost("Message", "Send text", "Queued and answered immediately with 202: the anti-ban layer (typing indicator, human delay, per-minute limit, one message at a time per chat) runs in the background. Failures after 3 attempts are reported to the webhook as `send_failed`.", bodyObj(map[string]any{
			"number": strProp("Phone number with country code, or a full JID", "5511999999999"),
			"text":   strProp("Message text", "Hello from Arcanum!"),
		})),
		"/api/message/sendMedia/{instance}": pathPost("Message", "Send media", "Image, audio, video or document. `media` must be the file in base64 (URLs are not downloaded by the gateway).", bodyObj(map[string]any{
			"number":    strProp("Recipient", "5511999999999"),
			"mediatype": strProp("image, audio, video or document (default)", "image"),
			"mimetype":  strProp("MIME type", "image/jpeg"),
			"caption":   strProp("Caption (image, video, document)", "Your receipt"),
			"media":     strProp("File content in base64", "<base64>"),
			"fileName":  strProp("File name shown for documents", "receipt.pdf"),
		})),
		"/api/message/sendLocation/{instance}": pathPost("Message", "Send location", "Send a map pin.", bodyObj(map[string]any{
			"number":    strProp("Recipient", "5511999999999"),
			"name":      strProp("Place name", "Head office"),
			"address":   strProp("Address", "Av. Paulista, 1000"),
			"latitude":  numProp("Latitude", -23.5614),
			"longitude": numProp("Longitude", -46.6559),
		})),
		"/api/message/sendContact/{instance}": pathPost("Message", "Send contact", "Send one or more contact cards (vCard).", bodyObj(map[string]any{
			"number":  strProp("Recipient", "5511999999999"),
			"contact": arrProp("Contacts", []string{`{"fullName":"Support","phoneNumber":"5511888888888"}`}),
		})),
		"/api/message/sendReaction/{instance}": pathPost("Message", "Send reaction", "React to a message with an emoji (empty string removes the reaction).", bodyObj(map[string]any{
			"key": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"remoteJid": strProp("Chat JID", "5511999999999@s.whatsapp.net"),
					"id":        strProp("Message ID (from the webhook)", "3EB0C431C26A1916E2F5"),
				},
			},
			"reaction": strProp("Emoji", "👍"),
		})),
		"/api/message/sendPoll/{instance}": pathPost("Message", "Send poll", "Send a poll.", bodyObj(map[string]any{
			"number":          strProp("Recipient", "5511999999999"),
			"name":            strProp("Question", "Best day for the meeting?"),
			"selectableCount": numProp("How many options can be picked", 1),
			"values":          arrProp("Options", []string{"Monday", "Tuesday", "Wednesday"}),
		})),

		// ── Chat ──
		"/api/chat/markMessageAsRead/{instance}": pathPost("Chat", "Mark as read", "Send read receipts (blue ticks).", bodyObj(map[string]any{
			"readMessages": arrProp("Messages", []string{`{"remoteJid":"5511999999999@s.whatsapp.net","id":"3EB0C431C26A1916E2F5"}`}),
		})),
		"/api/chat/checkWhatsAppNumbers/{instance}": pathPost("Chat", "Check numbers", "Which numbers have a WhatsApp account.", bodyObj(map[string]any{
			"numbers": arrProp("Phone numbers", []string{"5511999999999", "5511888888888"}),
		})),
		"/api/chat/updateProfileName/{instance}": pathPost("Chat", "Update profile name", "Change the display name of the account.", bodyObj(map[string]any{
			"name": strProp("New display name", "Acme Support"),
		})),
		"/api/chat/updateProfileStatus/{instance}": pathPost("Chat", "Update about", "Change the 'about' text.", bodyObj(map[string]any{
			"status": strProp("New about text", "Replies in minutes"),
		})),
		"/api/chat/updateProfilePicture/{instance}": pathPost("Chat", "Update profile picture", "Change the profile photo.", bodyObj(map[string]any{
			"picture": strProp("JPEG image in base64", "<base64>"),
		})),
		"/api/chat/archiveChat/{instance}": pathPost("Chat", "Archive chat (planned)", "Not implemented yet: returns 501.", bodyObj(map[string]any{
			"chat":    strProp("Chat JID", "5511999999999@s.whatsapp.net"),
			"archive": boolProp("Archive (true) or unarchive (false)", true),
		})),
		"/api/chat/findChats/{instance}":    pathPost("Chat", "Find chats (planned)", "Planned: needs a message store. Returns an empty list today.", nil),
		"/api/chat/findContacts/{instance}": pathPost("Chat", "Find contacts (planned)", "Planned: needs a contact store. Returns an empty list today.", nil),
		"/api/chat/findMessages/{instance}": pathPost("Chat", "Find messages (planned)", "Planned: needs a message store. Returns an empty list today.", nil),

		// ── Events ──
		"/api/events/webhook/{instance}": map[string]any{
			"get": op("Events", "Get webhook", "Current webhook configuration of the session."),
			"post": opBody("Events", "Set webhook", "Change the webhook URL, events or turn it on/off. The URL must be http/https; cloud metadata and link-local addresses are refused.", bodyObj(map[string]any{
				"url":     strProp("Webhook URL", "https://example.com/webhooks/whatsapp"),
				"events":  arrProp("Events to subscribe", []string{"messages", "status"}),
				"enabled": boolProp("Deliver events", true),
			})),
		},
		"/api/events/webhook/{instance}/test":       pathPost("Events", "Test webhook", "Send a signed `{\"event\":\"test\"}` delivery to the webhook and return the result (status, duration, error).", nil),
		"/api/events/webhook/{instance}/deliveries": pathGetParam("Events", "Webhook deliveries", "Last 100 deliveries of this session: event, HTTP status, duration, error. Bodies are not stored.", "instance"),
		"/api/events/webhooks/export":               pathGet("Events", "Export webhooks", "Webhook configuration of every session as a JSON file."),
		"/api/events/webhooks/import": map[string]any{"post": opBody("Events", "Import webhooks", "Apply an exported file to existing sessions. Unknown sessions and invalid URLs are skipped with the reason.", bodyObj(map[string]any{
			"webhooks": map[string]any{"type": "object", "description": "instance name -> {url, events, enabled}"},
		}))},

		// ── Settings ──
		"/api/settings/{instance}": map[string]any{
			"get": op("Settings", "Get settings", "Current settings of the session."),
			"post": opBody("Settings", "Update settings", "Today only `groupsIgnore` changes behaviour; the other flags are stored for upcoming releases.", bodyObj(map[string]any{
				"groupsIgnore":    boolProp("Do not forward group messages", true),
				"rejectCalls":     boolProp("Planned: auto-reject calls", false),
				"alwaysOnline":    boolProp("Planned: always online", false),
				"readMessages":    boolProp("Planned: auto read receipts", true),
				"readStatus":      boolProp("Planned: auto read status", false),
				"syncFullHistory": boolProp("Planned: full history sync", false),
			})),
		},
	}
}

// public tira a exigencia de chave de uma rota na spec (o middleware faz o mesmo).
func public(path map[string]any) map[string]any {
	for _, v := range path {
		if o, ok := v.(map[string]any); ok {
			o["security"] = []any{}
		}
	}
	return path
}

// ── helpers ──

func strProp(desc, example string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "example": example}
}
func numProp(desc string, example float64) map[string]any {
	return map[string]any{"type": "number", "description": desc, "example": example}
}
func boolProp(desc string, example bool) map[string]any {
	return map[string]any{"type": "boolean", "description": desc, "example": example}
}
func arrProp(desc string, example []string) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": map[string]string{"type": "string"}, "example": example}
}

func bodyObj(props map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"type": "object", "properties": props},
			},
		},
	}
}

func op(tag, summary, desc string) map[string]any {
	return map[string]any{
		"tags": []string{tag}, "summary": summary, "description": desc,
		"responses": map[string]any{
			"200": map[string]string{"description": "Success"},
			"401": map[string]string{"description": "Missing or invalid API key"},
		},
	}
}

func opBody(tag, summary, desc string, body map[string]any) map[string]any {
	o := op(tag, summary, desc)
	o["requestBody"] = body
	return o
}

func pathGet(tag, summary, desc string) map[string]any {
	return map[string]any{"get": op(tag, summary, desc)}
}

func pathPost(tag, summary, desc string, body map[string]any) map[string]any {
	o := op(tag, summary, desc)
	o["parameters"] = []map[string]any{instanceParam()}
	if body != nil {
		o["requestBody"] = body
	}
	if tag == "Instance" && summary == "Create instance" {
		delete(o, "parameters") // unica rota POST sem {instance} no caminho
	}
	return map[string]any{"post": o}
}

func instanceParam() map[string]any {
	return map[string]any{
		"name": "instance", "in": "path", "required": true,
		"schema": map[string]string{"type": "string"}, "description": "Instance name",
	}
}

func formatParam() map[string]any {
	return map[string]any{
		"name": "format", "in": "query", "required": false,
		"schema":      map[string]any{"type": "string", "enum": []string{"png"}},
		"description": "Set to 'png' to get the QR code as a binary PNG image instead of JSON",
	}
}

// pathGetQR documents GET endpoints backed by writeQRResponse: they return the
// QR code as a JSON data URL by default, or a binary PNG with ?format=png.
func pathGetQR(tag, summary, desc string) map[string]any {
	o := op(tag, summary, desc)
	o["parameters"] = []map[string]any{instanceParam(), formatParam()}
	o["responses"] = map[string]any{
		"200": map[string]any{
			"description": "QR code as JSON (data URL), or a binary PNG with ?format=png",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"instanceName": strProp("Instance name", "sales"),
							"status":       strProp("Connection status", "qr"),
							"qr":           strProp("QR code as a data URL", "data:image/png;base64,..."),
						},
					},
				},
				"image/png": map[string]any{
					"schema": map[string]any{"type": "string", "format": "binary"},
				},
			},
		},
		"401": map[string]string{"description": "Missing or invalid API key"},
	}
	return map[string]any{"get": o}
}

func pathPostParam(tag, summary, desc, _ string) map[string]any {
	o := op(tag, summary, desc)
	o["parameters"] = []map[string]any{instanceParam()}
	return map[string]any{"post": o}
}

func pathDelParam(tag, summary, desc, _ string) map[string]any {
	o := op(tag, summary, desc)
	o["parameters"] = []map[string]any{instanceParam()}
	return map[string]any{"delete": o}
}

func pathGetParam(tag, summary, desc, _ string) map[string]any {
	o := op(tag, summary, desc)
	o["parameters"] = []map[string]any{instanceParam()}
	return map[string]any{"get": o}
}
