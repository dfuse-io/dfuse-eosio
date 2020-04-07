caches.keys().then(function(cacheNames) {
  return Promise.all(
    cacheNames.map(function(cacheName) {
      // If this cache name isn't present in the array of "expected" cache names, then delete it.
      console.log("Deleting out of date cache:", cacheName)
      return caches.delete(cacheName)
    })
  )
})
