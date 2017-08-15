# About Splash
Splash gives you a managed resource pool. Resource pools provide a way to share exclusive resources among multiple goroutines in a safe way. Each pool uses a channel under the hood, and has a configured maximum capacity. 

Pools have the following capabilities:

 * Automatically maintain a minimum number of pool resources.
 * Automatically remove a resource when returning a resource causes the pool to exceed it's capacity.
 * Perform healthcheck on resources. Resources that fail healthchecks are removed.
 * If a pool connection has not been used for a length of time (ie, timeout), it will be removed.
 * Tack how many times the resources have been used, and for how long.

 ## Usage

