// SPDX-FileCopyrightText: 2026 HumanoidSandvichDispenser <humanoidsandvichdispenser@gmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;

use anyhow::{Context, Result, bail};
use teamtype::config::AppConfig;
use teamtype::daemon::DocumentActorHandle;
use teamtype::peer::ConnectionManager;
use teamtype::wormhole::put_secret_address_into_wormhole;
use tokio::sync::Mutex;

use crate::config::ServerConfig;

// One project == one base_dir == one iroh node identity.
pub type ProjectId = String;

/// A live project: document actor + iroh peer (no editor socket).
pub struct Project {
    pub id: ProjectId,
    pub base_dir: PathBuf,
    pub document: DocumentActorHandle,
    pub connection: ConnectionManager,
}

impl Project {
    // <node_id>#<passphrase>; joinable as-is.
    #[must_use]
    pub fn secret_address(&self) -> &str {
        self.connection.secret_address()
    }
}

// One live Project per id, brought up lazily on first request.
pub struct ProjectRegistry {
    config: ServerConfig,
    projects: Mutex<HashMap<ProjectId, Arc<Project>>>,
}

impl ProjectRegistry {
    #[must_use]
    pub fn new(config: ServerConfig) -> Self {
        Self {
            config,
            projects: Mutex::new(HashMap::new()),
        }
    }

    pub async fn open(&self, id: &str) -> Result<Arc<Project>> {
        validate_id(id)?;

        let mut projects = self.projects.lock().await;
        if let Some(project) = projects.get(id) {
            return Ok(project.clone());
        }

        let project = Arc::new(self.bring_up(id).await?);
        projects.insert(id.to_string(), project.clone());
        Ok(project)
    }

    async fn bring_up(&self, id: &str) -> Result<Project> {
        let base_dir = self.config.storage_root.join(id);
        // get_keypair writes <base_dir>/.teamtype/key, so the dir must exist.
        std::fs::create_dir_all(base_dir.join(".teamtype"))
            .with_context(|| format!("creating project dir for {id}"))?;

        let is_new = !base_dir.join(".teamtype/doc").exists();

        let app_config = AppConfig {
            base_dir: base_dir.clone(),
            iroh_relay: self.config.iroh_relay.clone(),
            magic_wormhole_relay: self.config.magic_wormhole_relay.clone(),
            ..AppConfig::default()
        };

        // args: init, is_host, persist
        let document = DocumentActorHandle::new(&app_config, is_new, true, true);

        let connection = ConnectionManager::new(&app_config, document.clone(), &base_dir)
            .await
            .with_context(|| format!("binding endpoint for {id}"))?;

        tracing::info!(project = id, address = connection.secret_address(), "project up");

        Ok(Project {
            id: id.to_string(),
            base_dir,
            document,
            connection,
        })
    }

    // TODO: put_secret_address_into_wormhole only logs the code; make it return
    // the Code so we can surface it. For now returns the secret address.
    pub async fn mint_join_code(&self, id: &str) -> Result<String> {
        let project = self.open(id).await?;
        put_secret_address_into_wormhole(
            project.secret_address(),
            self.config.magic_wormhole_relay.clone(),
        )
        .await;
        Ok(project.secret_address().to_string())
    }
}

// Reject ids that could escape the storage root.
fn validate_id(id: &str) -> Result<()> {
    let ok = !id.is_empty()
        && id.len() <= 64
        && id
            .chars()
            .all(|c| c.is_ascii_alphanumeric() || c == '-' || c == '_');
    if !ok {
        bail!("invalid project id: {id:?}");
    }
    Ok(())
}
