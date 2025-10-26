//! SAGE Smart Contract Rust Bindings
//! Generated: 2025-08-23T06:06:30.987Z

pub mod contracts {
    use ethers::prelude::*;

    // Generate bindings for SageRegistryV2
    abigen!(
        SageRegistryV2,
        "./abi/SageRegistryV2.json",
        event_derives(serde::Deserialize, serde::Serialize)
    );

    // Generate bindings for SageVerificationHook
    abigen!(
        SageVerificationHook,
        "./abi/SageVerificationHook.json",
        event_derives(serde::Deserialize, serde::Serialize)
    );
}

pub use contracts::*;

/// Example client for interacting with SAGE contracts
pub mod client {
    use super::*;
    use ethers::prelude::*;
    use std::sync::Arc;

    pub struct SageClient<M: Middleware> {
        pub registry: SageRegistryV2<M>,
        pub hook: Option<SageVerificationHook<M>>,
        client: Arc<M>,
    }

    impl<M: Middleware> SageClient<M> {
        /// Create a new client with registry address
        pub fn new(registry_address: Address, client: Arc<M>) -> Self {
            let registry = SageRegistryV2::new(registry_address, client.clone());
            Self {
                registry,
                hook: None,
                client,
            }
        }

        /// Add verification hook contract
        pub fn with_hook(mut self, hook_address: Address) -> Self {
            self.hook = Some(SageVerificationHook::new(hook_address, self.client.clone()));
            self
        }

        /// Get agent by DID
        pub async fn get_agent_by_did(&self, did: String) -> Result<contracts::AgentMetadata, ContractError<M>> {
            self.registry.get_agent_by_did(did).call().await
        }

        /// Register a new agent (requires signer)
        pub async fn register_agent(
            &self,
            did: String,
            name: String,
            description: String,
            endpoint: String,
            public_key: Bytes,
            capabilities: String,
            signature: Bytes,
        ) -> Result<TransactionReceipt, ContractError<M>> {
            let tx = self.registry
                .register_agent(did, name, description, endpoint, public_key, capabilities, signature);
            
            let pending = tx.send().await?;
            let receipt = pending.await?.ok_or_else(|| {
                ContractError::ProviderError(ProviderError::CustomError("Transaction failed".into()))
            })?;
            
            Ok(receipt)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_contract_generation() {
        // This will fail to compile if the abigen! macro fails
        let _ = std::mem::size_of::<contracts::SageRegistryV2<ethers::providers::Provider<ethers::providers::Http>>>();
    }
}
