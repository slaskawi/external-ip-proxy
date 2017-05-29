package org.infinispan.benchmark.cloud;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.net.SocketAddress;
import java.util.HashMap;
import java.util.Map;

import org.infinispan.client.hotrod.impl.AddressMapper;

/**
 * Created by slaskawi on 5/26/17.
 */
public class CloudAddressMapper implements AddressMapper {

   private DiscoveryInfo discoveryInfo;

   private Map<String, String> externalToInternalAddressesMapping = new HashMap<>();

   public CloudAddressMapper() {

      try {
         ClusterDiscoveryAgent agent = new ClusterDiscoveryAgent("http://35.187.73.162:8888");
         discoveryInfo = agent.discover();
         System.out.println("discoveryInfo = " + discoveryInfo);
      } catch (IOException e) {
         throw new RuntimeException(e);
      }
   }


   @Override
   public SocketAddress toExternalAddress(SocketAddress internalAddress) {
      InetSocketAddress socketAddress = ((InetSocketAddress) internalAddress);
      String narmalizedAddress = socketAddress.getHostName();
      String externalAddress = discoveryInfo.getExternalAddress(narmalizedAddress);
      if (externalAddress == null) {
         throw new IllegalStateException("No address mapping for " + internalAddress);
      }
      return new InetSocketAddress(externalAddress, socketAddress.getPort());
   }

   @Override
   public SocketAddress toInternalAddress(SocketAddress externalAddress) {
      throw new UnsupportedOperationException("not implemented");
   }
}
