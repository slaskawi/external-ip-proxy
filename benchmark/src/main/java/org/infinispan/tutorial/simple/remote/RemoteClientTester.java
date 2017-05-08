package org.infinispan.tutorial.simple.remote;

import org.infinispan.client.hotrod.RemoteCache;
import org.infinispan.client.hotrod.RemoteCacheManager;
import org.infinispan.client.hotrod.configuration.ClientIntelligence;
import org.infinispan.client.hotrod.configuration.ConfigurationBuilder;

public class RemoteClientTester {

   private static RemoteCacheManager cacheManager;
   private static RemoteCache<String, String> cache;


   public static final void main(String... args) {
      ConfigurationBuilder builder = new ConfigurationBuilder();
      builder.clientIntelligence(ClientIntelligence.HASH_DISTRIBUTION_AWARE);
      builder
            .addServer()
            .host("172.29.0.2").port(11222)
            .host("172.29.0.3").port(11222)
            .host("172.29.0.4").port(11222)
//            .host("172.17.0.2").port(11222)
//            .host("172.17.0.3").port(11222)
//            .host("172.17.0.5").port(11222)
            .build();
      cacheManager = new RemoteCacheManager(builder.build());
      cache = cacheManager.getCache();
      cache.clear();

      cache.put("test", "test");
      System.out.println(cache.get("test"));

   }

}
