package main

import "strings"

func openAPISpec() map[string]any {
	paths := map[string]any{
		"/api/health": map[string]any{
			"get": publicOperation("health", "服务健康状态", objectResponse("Go 服务、Agent TCP 与配置状态")),
		},
		"/api/schema": map[string]any{
			"get": publicOperation("schema", "ServerStatus API 简表", objectResponse("端点和集合描述")),
		},
		"/api/openapi.json": map[string]any{
			"get": publicOperation("openapi", "OpenAPI 3.1 文档", objectResponse("OpenAPI document")),
		},
		"/json/stats.json": map[string]any{
			"get": publicOperation("stats", "实时状态快照", objectResponse("WebUI 状态数据")),
		},
		"/api/config": map[string]any{
			"get": protectedOperation("getConfig", "读取完整配置", nil, configResponse("200", "配置文档")),
			"put": protectedOperation("replaceConfig", "整体校验并替换配置", requestBody("Config"), configResponse("200", "已保存并热重载的配置")),
		},
		"/api/reload": map[string]any{
			"post": protectedOperation("reloadConfig", "从磁盘热重载配置", nil, objectResponse("重载结果")),
		},
		"/api/restart": map[string]any{
			"post": protectedOperation("restartRuntime", "在进程内重启采集运行时", nil, statusResponse("202", "重启结果", map[string]any{
				"operation": map[string]any{"type": "string", "const": "restart"},
				"mode":      map[string]any{"type": "string", "const": "in-process"},
			})),
		},
		"/api/servers/{username}/reset-traffic": map[string]any{
			"post": withParameters(protectedOperation("resetServerTraffic", "重置节点本月流量基线", nil, objectResponse("流量重置结果")), pathParameter("username", "节点用户名")),
		},
	}

	for _, key := range []string{"servers", "monitors", "sslcerts", "watchdog"} {
		spec := collectionSpecs[key]
		schemaName := map[string]string{"servers": "Server", "monitors": "Monitor", "sslcerts": "SSLCert", "watchdog": "Watchdog"}[key]
		operationBase := strings.TrimSuffix(key, "s")
		if key == "sslcerts" {
			operationBase = "sslcert"
		}
		paths["/api/"+key] = map[string]any{
			"get":  protectedOperation("list"+titleWord(key), "查询 "+key, nil, collectionResponse(key, schemaName, "配置集合")),
			"post": protectedOperation("create"+titleWord(operationBase), "新增 "+spec.itemName, requestBody(schemaName), itemResponse("201", spec.itemName, schemaName, "创建结果")),
		}
		parameterName := "id"
		parameterDescription := "数字下标或唯一的 " + spec.idField
		if key == "servers" {
			parameterName = "username"
			parameterDescription = "节点用户名"
		}
		paths["/api/"+key+"/{"+parameterName+"}"] = map[string]any{
			"put":    withParameters(protectedOperation("update"+titleWord(operationBase), "修改 "+spec.itemName, requestBody(schemaName), itemResponse("200", spec.itemName, schemaName, "修改结果")), pathParameter(parameterName, parameterDescription)),
			"delete": withParameters(protectedOperation("delete"+titleWord(operationBase), "删除 "+spec.itemName, nil, itemResponse("200", "removed", schemaName, "删除结果")), pathParameter(parameterName, parameterDescription)),
		}
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "ServerStatus HTTP API",
			"version":     version,
			"description": "单进程 Go ServerStatus 的配置、运行状态和采集控制 API。",
		},
		"servers": []any{map[string]any{"url": "/", "description": "当前 ServerStatus 实例"}},
		"paths":   paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "ADMIN_TOKEN"},
			},
			"schemas": openAPISchemas(),
		},
	}
}

func openAPISchemas() map[string]any {
	stringProperty := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	integerProperty := func(description string, minimum, maximum int) map[string]any {
		property := map[string]any{"type": "integer", "description": description, "minimum": minimum}
		if maximum > 0 {
			property["maximum"] = maximum
		}
		return property
	}
	server := map[string]any{
		"type": "object", "required": collectionSpecs["servers"].required,
		"properties": map[string]any{
			"username": stringProperty("唯一客户端用户名"), "name": stringProperty("节点显示名称"),
			"type": stringProperty("虚拟化或节点类型"), "host": stringProperty("主机标识"),
			"location": stringProperty("位置"), "password": stringProperty("客户端密码"),
			"monthstart": integerProperty("月流量重置日", 1, 28), "disabled": map[string]any{"type": "boolean", "default": false},
		},
	}
	monitor := map[string]any{
		"type": "object", "required": collectionSpecs["monitors"].required,
		"properties": map[string]any{
			"name": stringProperty("监测名称"), "host": stringProperty("HTTP(S) URL 或 TCP 地址"),
			"type": stringProperty("https、http 或 tcp"), "interval": integerProperty("客户端探测间隔秒数", 1, 0),
		},
	}
	sslcert := map[string]any{
		"type": "object", "required": collectionSpecs["sslcerts"].required,
		"properties": map[string]any{
			"name": stringProperty("证书名称"), "domain": stringProperty("域名或 URL"),
			"port": integerProperty("TLS 端口", 1, 65535), "interval": integerProperty("检查间隔秒数", 1, 0),
			"callback": stringProperty("告警回调 URL 前缀"),
		},
	}
	watchdog := map[string]any{
		"type": "object", "required": collectionSpecs["watchdog"].required,
		"properties": map[string]any{
			"name": stringProperty("告警名称"), "rule": stringProperty("兼容 Exprtk 的状态表达式"),
			"interval": integerProperty("通知冷却秒数", 1, 0), "callback": stringProperty("告警回调 URL 前缀"),
		},
	}
	return map[string]any{
		"Server": server, "Monitor": monitor, "SSLCert": sslcert, "Watchdog": watchdog,
		"Config": map[string]any{
			"type": "object", "required": []string{"servers", "monitors", "sslcerts", "watchdog"},
			"properties": map[string]any{
				"servers": arraySchema("Server"), "monitors": arraySchema("Monitor"),
				"sslcerts": arraySchema("SSLCert"), "watchdog": arraySchema("Watchdog"),
			},
		},
		"Error": map[string]any{
			"type": "object", "required": []string{"ok", "error"},
			"properties": map[string]any{"ok": map[string]any{"type": "boolean", "const": false}, "error": map[string]any{"type": "string"}, "details": map[string]any{}},
		},
	}
}

func titleWord(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}
func arraySchema(name string) map[string]any {
	return map[string]any{"type": "array", "items": schemaRef(name)}
}

func requestBody(schemaName string) map[string]any {
	return map[string]any{
		"required": true,
		"content":  map[string]any{"application/json": map[string]any{"schema": schemaRef(schemaName)}},
	}
}

func objectResponse(description string) map[string]any {
	return map[string]any{"200": jsonResponse(description, map[string]any{"type": "object"}), "4XX": jsonResponse("请求错误", schemaRef("Error"))}
}

func configResponse(status, description string) map[string]any {
	return statusResponse(status, description, map[string]any{"config": schemaRef("Config")})
}

func collectionResponse(key, schemaName, description string) map[string]any {
	return statusResponse("200", description, map[string]any{key: arraySchema(schemaName)})
}

func itemResponse(status, key, schemaName, description string) map[string]any {
	return statusResponse(status, description, map[string]any{
		key:        schemaRef(schemaName),
		"reloaded": map[string]any{"type": "boolean", "const": true},
		"pid":      map[string]any{"type": "integer", "minimum": 1},
		"config":   schemaRef("Config"),
	})
}

func statusResponse(status, description string, properties map[string]any) map[string]any {
	required := []string{"ok"}
	allProperties := map[string]any{"ok": map[string]any{"type": "boolean", "const": true}}
	for key, property := range properties {
		allProperties[key] = property
		required = append(required, key)
	}
	return map[string]any{
		status: jsonResponse(description, map[string]any{
			"type": "object", "required": required, "properties": allProperties,
		}),
		"4XX": jsonResponse("请求错误", schemaRef("Error")),
	}
}

func jsonResponse(description string, schema map[string]any) map[string]any {
	return map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
}

func publicOperation(operationID, summary string, responses map[string]any) map[string]any {
	return map[string]any{"operationId": operationID, "summary": summary, "security": []any{}, "responses": responses}
}

func protectedOperation(operationID, summary string, body map[string]any, responses map[string]any) map[string]any {
	operation := map[string]any{
		"operationId": operationID, "summary": summary,
		"security": []any{map[string]any{"bearerAuth": []any{}}}, "responses": responses,
	}
	if body != nil {
		operation["requestBody"] = body
	}
	return operation
}

func withParameters(operation map[string]any, parameters ...map[string]any) map[string]any {
	items := make([]any, 0, len(parameters))
	for _, parameter := range parameters {
		items = append(items, parameter)
	}
	operation["parameters"] = items
	return operation
}

func pathParameter(name, description string) map[string]any {
	return map[string]any{"name": name, "in": "path", "required": true, "description": description, "schema": map[string]any{"type": "string"}}
}
