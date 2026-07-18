class FreeProxyError(Exception):
    """Base exception for expected application failures."""


class ProviderError(FreeProxyError):
    """Raised when a public proxy provider cannot be queried or parsed."""


class ResourceNotFoundError(FreeProxyError):
    """Raised when an API resource does not exist."""


class NetworkOperationError(FreeProxyError):
    """Raised when a privileged network operation fails."""
