package org.infinispan.benchmark.cloud;

import java.io.IOException;

import org.infinispan.client.hotrod.RemoteCacheManager;
import org.infinispan.client.hotrod.configuration.Configuration;
import org.infinispan.client.hotrod.configuration.ConfigurationBuilder;

public class SanityTest {

   public static void main(String[] args) throws IOException {
      ConfigurationBuilder hotRodConfiguration = new ConfigurationBuilder();
      hotRodConfiguration.addServer()
            .host("104.155.113.51").port(11222)
            .addressMapping(CloudAddressMapper.class);

      Configuration configuration = hotRodConfiguration.build();
      RemoteCacheManager remoteCacheManager = new RemoteCacheManager(configuration);
      remoteCacheManager.getCache().put("test", "test");
      System.out.println(remoteCacheManager.getCache().get("test"));
   }
}
