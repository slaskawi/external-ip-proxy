package org.infinispan.tutorial.simple.remote;

import java.io.IOException;

import org.infinispan.client.hotrod.RemoteCacheManager;
import org.infinispan.client.hotrod.configuration.ConfigurationBuilder;

/**
 * Created by slaskawi on 5/22/17.
 */
public class ClusterDiscoveryTest {

   public static void main(String[] args) throws IOException {
      ClusterDiscoveryAgent agent = new ClusterDiscoveryAgent("http://localhost:8888");
      DiscoveryInfo discover = agent.discover();
      System.out.println(discover);


      ConfigurationBuilder hotRodConfiguration = new ConfigurationBuilder();

      for (String internalAddress : discover.getAddresses().keySet()) {
         String externalAddress = discover.getAddresses().get(internalAddress);

         hotRodConfiguration.addServer().host(externalAddress);
      }

      RemoteCacheManager remoteCacheManager = new RemoteCacheManager(hotRodConfiguration.build());
      remoteCacheManager.getCache().put("test", "test");
      remoteCacheManager.getCache().get("test");



   }

}
