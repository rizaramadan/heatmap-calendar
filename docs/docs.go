// Package docs provides swagger documentation for the Heatmap Internal API.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "title": "Heatmap Internal API",
        "description": "API for managing heatmap loads, entities, groups, and capacity tracking",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "email": "support@example.com"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "1.0"
    },
    "host": "{{.Host}}",
    "basePath": "/",
    "paths": {
        "/api/entities": {
            "get": {
                "description": "Returns all entities or filters by type (person/group)",
                "produces": ["application/json"],
                "tags": ["Entities"],
                "summary": "List all entities",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Filter by entity type (person or group)",
                        "name": "type",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "List of entities",
                        "schema": {
                            "type": "array",
                            "items": {"$ref": "#/definitions/Entity"}
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            },
            "post": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Create a new person or group entity",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Entities"],
                "summary": "Create a new entity",
                "parameters": [
                    {
                        "description": "Entity to create",
                        "name": "entity",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/CreateEntityRequest"}
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created entity",
                        "schema": {"$ref": "#/definitions/Entity"}
                    },
                    "400": {
                        "description": "Invalid request body",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/api/entities/{id}": {
            "get": {
                "description": "Returns a single entity by its ID",
                "produces": ["application/json"],
                "tags": ["Entities"],
                "summary": "Get entity by ID",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Entity ID (email for persons, string ID for groups)",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Entity details",
                        "schema": {"$ref": "#/definitions/Entity"}
                    },
                    "404": {
                        "description": "Entity not found",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            },
            "delete": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Delete an entity by its ID",
                "produces": ["application/json"],
                "tags": ["Entities"],
                "summary": "Delete an entity",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Entity ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Success message",
                        "schema": {"$ref": "#/definitions/SuccessResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "404": {
                        "description": "Entity not found",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/api/loads/upsert": {
            "post": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Create or update a load item with assignments (for n8n integration)",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Loads"],
                "summary": "Upsert a load",
                "parameters": [
                    {
                        "description": "Load data to upsert",
                        "name": "load",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/UpsertLoadRequest"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Success with load ID",
                        "schema": {"$ref": "#/definitions/UpsertLoadResponse"}
                    },
                    "400": {
                        "description": "Invalid request body",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/api/heatmap/{entity}": {
            "get": {
                "description": "Returns the heatmap grid partial for an entity",
                "produces": ["text/html"],
                "tags": ["Heatmap"],
                "summary": "Get heatmap partial for entity",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Entity ID",
                        "name": "entity",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "HTML partial for heatmap grid"
                    },
                    "500": {
                        "description": "Failed to load heatmap"
                    }
                }
            }
        },
        "/api/heatmap/{entity}/day/{date}": {
            "get": {
                "description": "Returns the tasks/loads for a specific day",
                "produces": ["text/html"],
                "tags": ["Heatmap"],
                "summary": "Get day details for entity",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Entity ID",
                        "name": "entity",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Date in YYYY-MM-DD format",
                        "name": "date",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "HTML partial for day details"
                    },
                    "400": {
                        "description": "Invalid date format"
                    },
                    "500": {
                        "description": "Failed to load day details"
                    }
                }
            }
        },
        "/api/groups/{id}/members": {
            "get": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Returns all members of a group",
                "produces": ["application/json"],
                "tags": ["Groups"],
                "summary": "Get group members",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Group ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Group members",
                        "schema": {"$ref": "#/definitions/GroupMembersResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            },
            "post": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Add a person to a group",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Groups"],
                "summary": "Add member to group",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Group ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "description": "Member to add",
                        "name": "member",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/AddGroupMemberRequest"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Success message",
                        "schema": {"$ref": "#/definitions/SuccessResponse"}
                    },
                    "400": {
                        "description": "Invalid request or entity is not a group/person",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "404": {
                        "description": "Group or person not found",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/api/groups/{id}/members/{member}": {
            "delete": {
                "security": [{"ApiKeyAuth": []}],
                "description": "Remove a person from a group",
                "produces": ["application/json"],
                "tags": ["Groups"],
                "summary": "Remove member from group",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Group ID",
                        "name": "id",
                        "in": "path",
                        "required": true
                    },
                    {
                        "type": "string",
                        "description": "Member email to remove",
                        "name": "member",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Success message",
                        "schema": {"$ref": "#/definitions/SuccessResponse"}
                    },
                    "401": {
                        "description": "Unauthorized - invalid or missing API key",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/api/my-capacity": {
            "post": {
                "security": [{"SessionAuth": []}],
                "description": "Update capacity settings for the currently logged-in user",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["Capacity"],
                "summary": "Update user capacity",
                "parameters": [
                    {
                        "description": "Capacity update request",
                        "name": "capacity",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/UpdateCapacityRequest"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Success message",
                        "schema": {"$ref": "#/definitions/SuccessResponse"}
                    },
                    "400": {
                        "description": "Invalid request",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "401": {
                        "description": "Not authenticated",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/auth/request-otp": {
            "post": {
                "description": "Send an OTP code to the user's email for authentication",
                "consumes": ["application/json", "application/x-www-form-urlencoded"],
                "produces": ["application/json", "text/html"],
                "tags": ["Authentication"],
                "summary": "Request OTP",
                "parameters": [
                    {
                        "description": "Email address",
                        "name": "email",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/OTPRequest"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OTP sent successfully",
                        "schema": {"$ref": "#/definitions/SuccessResponse"}
                    },
                    "400": {
                        "description": "Invalid email",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "404": {
                        "description": "Email not found",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Failed to send OTP",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/auth/verify-otp": {
            "post": {
                "description": "Verify the OTP and create a session",
                "consumes": ["application/json", "application/x-www-form-urlencoded"],
                "produces": ["application/json", "text/html"],
                "tags": ["Authentication"],
                "summary": "Verify OTP",
                "parameters": [
                    {
                        "description": "Email and OTP",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/VerifyOTPRequest"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OTP verified, session created"
                    },
                    "400": {
                        "description": "Invalid OTP",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    },
                    "500": {
                        "description": "Failed to create session",
                        "schema": {"$ref": "#/definitions/ErrorResponse"}
                    }
                }
            }
        },
        "/auth/logout": {
            "post": {
                "description": "End the current session",
                "produces": ["application/json"],
                "tags": ["Authentication"],
                "summary": "Logout",
                "responses": {
                    "302": {
                        "description": "Redirect to home page"
                    }
                }
            }
        }
    },
    "definitions": {
        "Entity": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "string",
                    "description": "Entity ID (email for persons, string-id for groups)",
                    "example": "john@example.com"
                },
                "title": {
                    "type": "string",
                    "description": "Display name",
                    "example": "John Doe"
                },
                "type": {
                    "type": "string",
                    "description": "Entity type",
                    "enum": ["person", "group"],
                    "example": "person"
                },
                "default_capacity": {
                    "type": "number",
                    "description": "Default daily capacity",
                    "example": 5.0
                },
                "created_at": {
                    "type": "string",
                    "format": "date-time",
                    "description": "Creation timestamp"
                }
            }
        },
        "CreateEntityRequest": {
            "type": "object",
            "required": ["id", "title", "type"],
            "properties": {
                "id": {
                    "type": "string",
                    "description": "Entity ID",
                    "example": "john@example.com"
                },
                "title": {
                    "type": "string",
                    "description": "Display name",
                    "example": "John Doe"
                },
                "type": {
                    "type": "string",
                    "description": "Entity type (person or group)",
                    "enum": ["person", "group"],
                    "example": "person"
                },
                "default_capacity": {
                    "type": "number",
                    "description": "Default daily capacity (defaults to 5.0)",
                    "example": 5.0
                }
            }
        },
        "UpsertLoadRequest": {
            "type": "object",
            "required": ["external_id", "title", "date", "assignees"],
            "properties": {
                "external_id": {
                    "type": "string",
                    "description": "External system ID for deduplication",
                    "example": "gcal-123456"
                },
                "title": {
                    "type": "string",
                    "description": "Load/task title",
                    "example": "Team Meeting"
                },
                "source": {
                    "type": "string",
                    "description": "Origin system (gcal, crm, etc.)",
                    "example": "gcal"
                },
                "date": {
                    "type": "string",
                    "description": "Date in YYYY-MM-DD format",
                    "example": "2026-01-19"
                },
                "assignees": {
                    "type": "array",
                    "description": "List of assignees with weights",
                    "items": {
                        "type": "object",
                        "required": ["email"],
                        "properties": {
                            "email": {
                                "type": "string",
                                "description": "Assignee email",
                                "example": "john@example.com"
                            },
                            "weight": {
                                "type": "number",
                                "description": "Weight of assignment (defaults to 1.0)",
                                "example": 1.0
                            }
                        }
                    }
                }
            }
        },
        "UpsertLoadResponse": {
            "type": "object",
            "properties": {
                "success": {
                    "type": "boolean",
                    "example": true
                },
                "load_id": {
                    "type": "integer",
                    "description": "ID of the created/updated load",
                    "example": 123
                }
            }
        },
        "UpdateCapacityRequest": {
            "type": "object",
            "properties": {
                "default_capacity": {
                    "type": "number",
                    "description": "New default capacity value",
                    "example": 6.0
                },
                "date_overrides": {
                    "type": "array",
                    "description": "Date-specific capacity overrides",
                    "items": {
                        "type": "object",
                        "required": ["date", "capacity"],
                        "properties": {
                            "date": {
                                "type": "string",
                                "description": "Date in YYYY-MM-DD format",
                                "example": "2026-01-20"
                            },
                            "capacity": {
                                "type": "number",
                                "description": "Capacity for that date",
                                "example": 3.0
                            }
                        }
                    }
                }
            }
        },
        "AddGroupMemberRequest": {
            "type": "object",
            "required": ["person_email"],
            "properties": {
                "person_email": {
                    "type": "string",
                    "description": "Email of the person to add",
                    "example": "john@example.com"
                }
            }
        },
        "GroupMembersResponse": {
            "type": "object",
            "properties": {
                "group_id": {
                    "type": "string",
                    "description": "Group ID",
                    "example": "engineering-team"
                },
                "members": {
                    "type": "array",
                    "description": "List of member emails",
                    "items": {
                        "type": "string"
                    },
                    "example": ["john@example.com", "jane@example.com"]
                }
            }
        },
        "OTPRequest": {
            "type": "object",
            "required": ["email"],
            "properties": {
                "email": {
                    "type": "string",
                    "format": "email",
                    "description": "User email address",
                    "example": "john@example.com"
                }
            }
        },
        "VerifyOTPRequest": {
            "type": "object",
            "required": ["email", "otp"],
            "properties": {
                "email": {
                    "type": "string",
                    "format": "email",
                    "description": "User email address",
                    "example": "john@example.com"
                },
                "otp": {
                    "type": "string",
                    "description": "6-digit OTP code",
                    "example": "123456"
                }
            }
        },
        "ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "description": "Error message",
                    "example": "invalid request body"
                }
            }
        },
        "SuccessResponse": {
            "type": "object",
            "properties": {
                "success": {
                    "type": "string",
                    "description": "Success message",
                    "example": "operation completed"
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "type": "apiKey",
            "name": "x-api-key",
            "in": "header",
            "description": "API Key for protected endpoints"
        },
        "SessionAuth": {
            "type": "apiKey",
            "name": "session",
            "in": "cookie",
            "description": "Session cookie for authenticated users"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Heatmap Internal API",
	Description:      "API for managing heatmap loads, entities, groups, and capacity tracking",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
