// SPDX-FileCopyrightText: 2026 HumanoidSandvichDispenser <humanoidsandvichdispenser@gmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

use std::sync::Arc;

use axum::Json;
use axum::extract::{Path, State};
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use axum::routing::{get, post};
use axum::Router;
use serde::Serialize;

use crate::registry::ProjectRegistry;

pub fn router(registry: Arc<ProjectRegistry>) -> Router {
    Router::new()
        .route("/projects/{id}/open", post(open_project))
        .route("/projects/{id}/secret-address", get(secret_address))
        .route("/projects/{id}/join-code", post(join_code))
        .with_state(registry)
}

#[derive(Serialize)]
struct ProjectInfo {
    id: String,
    secret_address: String,
}

#[derive(Serialize)]
struct JoinCode {
    secret_address: String,
}

async fn open_project(
    State(registry): State<Arc<ProjectRegistry>>,
    Path(id): Path<String>,
) -> Result<Json<ProjectInfo>, ApiError> {
    let project = registry.open(&id).await?;
    Ok(Json(ProjectInfo {
        id: project.id.clone(),
        secret_address: project.secret_address().to_string(),
    }))
}

async fn secret_address(
    State(registry): State<Arc<ProjectRegistry>>,
    Path(id): Path<String>,
) -> Result<Json<ProjectInfo>, ApiError> {
    open_project(State(registry), Path(id)).await
}

async fn join_code(
    State(registry): State<Arc<ProjectRegistry>>,
    Path(id): Path<String>,
) -> Result<Json<JoinCode>, ApiError> {
    let secret_address = registry.mint_join_code(&id).await?;
    Ok(Json(JoinCode { secret_address }))
}

// anyhow::Error -> 400.
struct ApiError(anyhow::Error);

impl From<anyhow::Error> for ApiError {
    fn from(err: anyhow::Error) -> Self {
        Self(err)
    }
}

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        tracing::warn!(error = %self.0, "request failed");
        (StatusCode::BAD_REQUEST, self.0.to_string()).into_response()
    }
}
