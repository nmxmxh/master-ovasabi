import React, { useState, useEffect } from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Heading,
  Badge,
  Spinner,
  Center,
  SimpleGrid,
  Icon,
  Code,
  Container
} from '@chakra-ui/react';
import {
  FiTarget,
  FiRefreshCw,
  FiCheckCircle,
  FiXCircle,
  FiClock,
  FiActivity
} from 'react-icons/fi';
import { useCampaignStore } from '../store/stores/campaignStore';
import { useCampaignData } from '../providers/CampaignProvider';
import { useMetadata } from '../store/hooks/useMetadata';
import { useEventHistory } from '../store/hooks/useEvents';
import CampaignUIRenderer from '../components/CampaignUIRenderer';
import './CampaignSwitchingPage.css';

interface CampaignSwitchEvent {
  old_campaign_id: string;
  new_campaign_id: string;
  reason: string;
  timestamp: string;
  status?: string;
}

const CampaignSwitchingPage: React.FC = () => {
  const { switchCampaignWithData, currentCampaign } = useCampaignStore();
  const {
    campaigns,
    loading: campaignsLoading,
    error: campaignsError,
    refresh: refreshCampaigns
  } = useCampaignData();
  const { metadata } = useMetadata();
  const events = useEventHistory(undefined, 50);
  // Simple toast implementation
  const showToast = (
    title: string,
    description: string,
    status: 'info' | 'success' | 'error' = 'info'
  ) => {
    console.log(`[Toast] ${status.toUpperCase()}: ${title} - ${description}`);
  };

  const [switchHistory, setSwitchHistory] = useState<CampaignSwitchEvent[]>([]);
  const [isSwitching, setIsSwitching] = useState(false);
  const [selectedCampaign, setSelectedCampaign] = useState<any>(null);
  const [showDetails, setShowDetails] = useState(false);
  const [activeTab, setActiveTab] = useState(0);

  // Filter campaign switch events
  const switchEvents = events.filter(
    e =>
      e.type?.includes('campaign:switch') ||
      e.type?.includes('campaign:switch:required') ||
      e.type?.includes('campaign:switch:completed')
  );

  // Update switch history when events change
  useEffect(() => {
    const newHistory: CampaignSwitchEvent[] = [];

    switchEvents.forEach(event => {
      if (event.type === 'campaign:switch:required' || event.type === 'campaign:switch:completed') {
        const payload = event.payload as any;
        if (payload) {
          newHistory.push({
            old_campaign_id: payload.old_campaign_id || 'Unknown',
            new_campaign_id: payload.new_campaign_id || 'Unknown',
            reason: payload.reason || 'Unknown',
            timestamp: event.timestamp || new Date().toISOString(),
            status: event.type.includes('completed') ? 'completed' : 'in_progress'
          });
        }
      }
    });

    setSwitchHistory(
      newHistory.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    );
  }, [switchEvents]);

  const handleCampaignSwitch = async (campaign: any) => {
    if (isSwitching || currentCampaign?.campaignId === campaign.id) {
      return;
    }

    setSelectedCampaign(campaign);
    setIsSwitching(true);

    showToast('Switching Campaign', `Switching to ${campaign.title || campaign.name}...`, 'info');

    try {
      if (switchCampaignWithData) {
        await new Promise<void>((resolve, reject) => {
          switchCampaignWithData(campaign, response => {
            if (response.type?.includes('success')) {
              console.log('Campaign switched successfully:', response);
              showToast(
                'Campaign Switched!',
                `Successfully switched to ${campaign.title || campaign.name}`,
                'success'
              );
              resolve();
            } else if (response.type?.includes('failed')) {
              console.error('Campaign switch failed:', response);
              showToast(
                'Switch Failed',
                response.payload?.error || 'Failed to switch campaign',
                'error'
              );
              reject(new Error(response.payload?.error || 'Switch failed'));
            } else {
              resolve();
            }
          });
        });
      }
    } catch (error) {
      console.error('Campaign switch error:', error);
      showToast(
        'Switch Error',
        error instanceof Error ? error.message : 'Unknown error occurred',
        'error'
      );
    } finally {
      setIsSwitching(false);
      setSelectedCampaign(null);
    }
  };

  const getCampaignStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'green';
      case 'inactive':
        return 'red';
      case 'draft':
        return 'yellow';
      default:
        return 'gray';
    }
  };

  const getSwitchStatusColor = (status: string) => {
    switch (status) {
      case 'completed':
        return 'green';
      case 'in_progress':
        return 'blue';
      case 'failed':
        return 'red';
      default:
        return 'gray';
    }
  };

  const getSwitchStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return FiCheckCircle;
      case 'in_progress':
        return FiRefreshCw;
      case 'failed':
        return FiXCircle;
      default:
        return FiClock;
    }
  };

  return (
    <Box minH="100vh" bg="gray.50">
      <Container maxW="1400px" py={6}>
        <VStack gap={8} align="stretch">
          {/* Header */}
          <Box textAlign="center" bg="white" p={8} borderRadius="lg" shadow="sm">
            <Heading size="2xl" className="gradient-text" mb={4}>
              <Icon as={FiTarget} mr={3} />
              Campaign Switching Interface
            </Heading>
            <Text fontSize="lg" color="gray.600">
              Real-time campaign management with seamless switching capabilities
            </Text>
          </Box>

          {/* Tab Navigation */}
          <Box bg="white" borderRadius="lg" shadow="sm" overflow="hidden">
            <HStack gap={0} borderBottom="1px" borderColor="gray.200">
              <Button
                variant={activeTab === 0 ? 'solid' : 'ghost'}
                colorScheme={activeTab === 0 ? 'blue' : 'gray'}
                borderRadius="none"
                onClick={() => setActiveTab(0)}
              >
                Campaign Interface
              </Button>
              <Button
                variant={activeTab === 1 ? 'solid' : 'ghost'}
                colorScheme={activeTab === 1 ? 'blue' : 'gray'}
                borderRadius="none"
                onClick={() => setActiveTab(1)}
              >
                Management
              </Button>
              <Button
                variant={activeTab === 2 ? 'solid' : 'ghost'}
                colorScheme={activeTab === 2 ? 'blue' : 'gray'}
                borderRadius="none"
                onClick={() => setActiveTab(2)}
              >
                History & Events
              </Button>
            </HStack>

            {/* Tab Content */}
            {activeTab === 0 && (
              <Box>
                <CampaignUIRenderer
                  campaign={currentCampaign}
                  isLoading={isSwitching && selectedCampaign !== null}
                />
              </Box>
            )}

            {activeTab === 1 && (
              <Box p={6}>
                <VStack gap={8} align="stretch">
                  {/* Current Campaign Status */}
                  <Box className="card-elevation" p={6} borderRadius="md" bg="white">
                    <Heading size="md" mb={4}>
                      Current Campaign Status
                    </Heading>
                    {currentCampaign ? (
                      <VStack gap={4} align="stretch">
                        <HStack justify="space-between">
                          <VStack align="start" gap={2}>
                            <Text fontSize="lg" fontWeight="bold">
                              {currentCampaign.title ||
                                currentCampaign.campaignName ||
                                'Unknown Campaign'}
                            </Text>
                            <HStack>
                              <Badge
                                colorScheme={getCampaignStatusColor(
                                  currentCampaign.status || 'unknown'
                                )}
                              >
                                {currentCampaign.status || 'Unknown'}
                              </Badge>
                              <Text fontSize="sm" color="gray.500">
                                ID: {currentCampaign.campaignId}
                              </Text>
                            </HStack>
                          </VStack>
                          <VStack align="end" gap={2}>
                            <Text fontSize="sm" color="gray.500">
                              Last Switched
                            </Text>
                            <Text fontSize="sm">
                              {currentCampaign.last_switched
                                ? new Date(currentCampaign.last_switched).toLocaleString()
                                : 'Never'}
                            </Text>
                          </VStack>
                        </HStack>

                        {currentCampaign.features && currentCampaign.features.length > 0 && (
                          <Box>
                            <Text fontSize="sm" fontWeight="medium" mb={2}>
                              Features:
                            </Text>
                            <HStack wrap="wrap" gap={2}>
                              {currentCampaign.features.map((feature: string, index: number) => (
                                <Badge key={index} colorScheme="blue" className="feature-tag">
                                  {feature}
                                </Badge>
                              ))}
                            </HStack>
                          </Box>
                        )}
                      </VStack>
                    ) : (
                      <Box p={4} bg="blue.50" borderRadius="md" border="1px" borderColor="blue.200">
                        <Text color="blue.800" fontWeight="medium">
                          No Active Campaign
                        </Text>
                        <Text color="blue.600" fontSize="sm">
                          Select a campaign to get started.
                        </Text>
                      </Box>
                    )}
                  </Box>

                  {/* System Stats */}
                  <SimpleGrid columns={{ base: 1, md: 4 }} gap={4}>
                    <Box
                      className="card-elevation"
                      p={4}
                      borderRadius="md"
                      bg="white"
                      textAlign="center"
                    >
                      <Text fontSize="2xl" fontWeight="bold" color="blue.500">
                        {campaigns.length}
                      </Text>
                      <Text fontSize="sm" color="gray.600">
                        Total Campaigns
                      </Text>
                    </Box>
                    <Box
                      className="card-elevation"
                      p={4}
                      borderRadius="md"
                      bg="white"
                      textAlign="center"
                    >
                      <Text fontSize="2xl" fontWeight="bold" color="green.500">
                        {campaigns.filter(c => c.status === 'active').length}
                      </Text>
                      <Text fontSize="sm" color="gray.600">
                        Active Campaigns
                      </Text>
                    </Box>
                    <Box
                      className="card-elevation"
                      p={4}
                      borderRadius="md"
                      bg="white"
                      textAlign="center"
                    >
                      <Text fontSize="2xl" fontWeight="bold" color="purple.500">
                        {switchHistory.length}
                      </Text>
                      <Text fontSize="sm" color="gray.600">
                        Switch Events
                      </Text>
                    </Box>
                    <Box
                      className="card-elevation"
                      p={4}
                      borderRadius="md"
                      bg="white"
                      textAlign="center"
                    >
                      <Text fontSize="2xl" fontWeight="bold" color="orange.500">
                        {events.length}
                      </Text>
                      <Text fontSize="sm" color="gray.600">
                        System Events
                      </Text>
                    </Box>
                  </SimpleGrid>

                  {/* Available Campaigns */}
                  <Box className="card-elevation" p={6} borderRadius="md" bg="white">
                    <HStack justify="space-between" mb={4}>
                      <Heading size="md">Available Campaigns</Heading>
                      <Button
                        onClick={refreshCampaigns}
                        size="sm"
                        variant="outline"
                        loading={campaignsLoading}
                      >
                        <Icon as={FiRefreshCw} mr={2} />
                        Refresh
                      </Button>
                    </HStack>

                    {campaignsLoading ? (
                      <Center py={8}>
                        <Spinner size="lg" />
                      </Center>
                    ) : campaignsError ? (
                      <Box p={4} bg="red.50" borderRadius="md" border="1px" borderColor="red.200">
                        <Text color="red.800" fontWeight="medium">
                          Error Loading Campaigns
                        </Text>
                        <Text color="red.600" fontSize="sm">
                          {campaignsError}
                        </Text>
                      </Box>
                    ) : (
                      <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap={4}>
                        {campaigns.map(campaign => (
                          <Box
                            key={campaign.id}
                            className={`card-elevation campaign-card-hover ${currentCampaign?.campaignId === campaign.id ? 'active' : ''}`}
                            p={4}
                            borderRadius="md"
                            bg="white"
                            border="2px"
                            borderColor={
                              currentCampaign?.campaignId === campaign.id ? 'blue.500' : 'gray.200'
                            }
                            cursor="pointer"
                            onClick={() => handleCampaignSwitch(campaign)}
                          >
                            <VStack gap={3} align="stretch">
                              <HStack justify="space-between">
                                <Text fontWeight="bold" fontSize="lg">
                                  {campaign.title || campaign.name}
                                </Text>
                                <Badge
                                  colorScheme={getCampaignStatusColor(campaign.status || 'unknown')}
                                >
                                  {campaign.status}
                                </Badge>
                              </HStack>

                              <Text fontSize="sm" color="gray.600" lineClamp={2}>
                                {campaign.description || 'No description available'}
                              </Text>

                              {campaign.features && campaign.features.length > 0 && (
                                <HStack wrap="wrap" gap={1}>
                                  {campaign.features
                                    .slice(0, 3)
                                    .map((feature: string, index: number) => (
                                      <Badge key={index} size="sm" colorScheme="blue">
                                        {feature}
                                      </Badge>
                                    ))}
                                  {campaign.features.length > 3 && (
                                    <Badge size="sm" colorScheme="gray">
                                      +{campaign.features.length - 3} more
                                    </Badge>
                                  )}
                                </HStack>
                              )}

                              <Button
                                colorScheme="blue"
                                size="sm"
                                onClick={e => {
                                  e.stopPropagation();
                                  handleCampaignSwitch(campaign);
                                }}
                                loading={isSwitching && selectedCampaign?.id === campaign.id}
                                disabled={
                                  isSwitching || currentCampaign?.campaignId === campaign.id
                                }
                              >
                                {currentCampaign?.campaignId === campaign.id
                                  ? 'Current Campaign'
                                  : 'Switch To'}
                              </Button>
                            </VStack>
                          </Box>
                        ))}
                      </SimpleGrid>
                    )}
                  </Box>

                  {/* Switch History */}
                  <Box className="card-elevation" p={6} borderRadius="md" bg="white">
                    <HStack justify="space-between" mb={4}>
                      <Heading size="md">Switch History</Heading>
                      <Button
                        onClick={() => setShowDetails(!showDetails)}
                        size="sm"
                        variant="outline"
                      >
                        <Icon as={FiActivity} mr={2} />
                        {showDetails ? 'Hide Details' : 'View Details'}
                      </Button>
                    </HStack>

                    {switchHistory.length === 0 ? (
                      <Box p={4} bg="blue.50" borderRadius="md" border="1px" borderColor="blue.200">
                        <Text color="blue.800" fontWeight="medium">
                          No Switch History
                        </Text>
                        <Text color="blue.600" fontSize="sm">
                          Campaign switches will appear here.
                        </Text>
                      </Box>
                    ) : (
                      <VStack gap={3} align="stretch">
                        {switchHistory
                          .slice(0, showDetails ? switchHistory.length : 5)
                          .map((switchEvent, index) => (
                            <Box
                              key={index}
                              className={`switch-history-item ${switchEvent.status || 'completed'}`}
                              p={4}
                              borderRadius="md"
                              bg="gray.50"
                            >
                              <HStack justify="space-between" mb={2}>
                                <HStack>
                                  <Icon
                                    as={getSwitchStatusIcon(switchEvent.status || 'completed')}
                                    color={getSwitchStatusColor(switchEvent.status || 'completed')}
                                  />
                                  <Text fontWeight="medium">
                                    {switchEvent.old_campaign_id} → {switchEvent.new_campaign_id}
                                  </Text>
                                </HStack>
                                <Badge
                                  colorScheme={getSwitchStatusColor(
                                    switchEvent.status || 'completed'
                                  )}
                                >
                                  {switchEvent.status || 'completed'}
                                </Badge>
                              </HStack>
                              <HStack justify="space-between">
                                <Text fontSize="sm" color="gray.600">
                                  Reason: {switchEvent.reason}
                                </Text>
                                <Text fontSize="sm" color="gray.500">
                                  {new Date(switchEvent.timestamp).toLocaleString()}
                                </Text>
                              </HStack>
                            </Box>
                          ))}
                      </VStack>
                    )}
                  </Box>

                  {/* System Information */}
                  <Box className="card-elevation" p={6} borderRadius="md" bg="white">
                    <Heading size="md" mb={4}>
                      System Information
                    </Heading>

                    <Box mb={4}>
                      <Text fontWeight="medium" mb={2}>
                        Metadata
                      </Text>
                      <Code
                        p={4}
                        display="block"
                        whiteSpace="pre-wrap"
                        fontSize="sm"
                        bg="gray.100"
                        borderRadius="md"
                      >
                        {JSON.stringify(metadata, null, 2)}
                      </Code>
                    </Box>

                    <Box>
                      <Text fontWeight="medium" mb={2}>
                        Recent Events ({events.length})
                      </Text>
                      <Box maxH="300px" overflowY="auto" bg="gray.50" p={4} borderRadius="md">
                        <VStack gap={2} align="stretch">
                          {events.slice(0, 10).map((event, index) => (
                            <HStack
                              key={index}
                              justify="space-between"
                              p={2}
                              bg="white"
                              borderRadius="sm"
                            >
                              <Text fontSize="xs" lineClamp={1} flex={1}>
                                {event.type}
                              </Text>
                              <Text fontSize="xs" color="gray.500">
                                {new Date(event.timestamp).toLocaleTimeString()}
                              </Text>
                              <Badge
                                size="sm"
                                colorScheme={event.type?.includes('success') ? 'green' : 'gray'}
                              >
                                {event.type?.includes('success') ? 'Success' : 'Info'}
                              </Badge>
                            </HStack>
                          ))}
                        </VStack>
                      </Box>
                    </Box>
                  </Box>
                </VStack>
              </Box>
            )}

            {activeTab === 2 && (
              <Box p={6}>
                <VStack gap={8} align="stretch">
                  {/* Switch History */}
                  <Box bg="white" p={6} borderRadius="lg" shadow="sm">
                    <HStack justify="space-between" mb={4}>
                      <Heading size="md">Switch History</Heading>
                      <Button
                        onClick={() => setShowDetails(!showDetails)}
                        size="sm"
                        variant="outline"
                      >
                        <Icon as={FiActivity} mr={2} />
                        {showDetails ? 'Hide Details' : 'View Details'}
                      </Button>
                    </HStack>

                    {switchHistory.length === 0 ? (
                      <Box p={4} bg="blue.50" borderRadius="md" border="1px" borderColor="blue.200">
                        <Text color="blue.800" fontWeight="medium">
                          No Switch History
                        </Text>
                        <Text color="blue.600" fontSize="sm">
                          Campaign switches will appear here.
                        </Text>
                      </Box>
                    ) : (
                      <VStack gap={3} align="stretch">
                        {switchHistory
                          .slice(0, showDetails ? switchHistory.length : 5)
                          .map((switchEvent, index) => (
                            <Box
                              key={index}
                              className={`switch-history-item ${switchEvent.status || 'completed'}`}
                              p={4}
                              borderRadius="md"
                              bg="gray.50"
                            >
                              <HStack justify="space-between" mb={2}>
                                <HStack>
                                  <Icon
                                    as={getSwitchStatusIcon(switchEvent.status || 'completed')}
                                    color={getSwitchStatusColor(switchEvent.status || 'completed')}
                                  />
                                  <Text fontWeight="medium">
                                    {switchEvent.old_campaign_id} → {switchEvent.new_campaign_id}
                                  </Text>
                                </HStack>
                                <Badge
                                  colorScheme={getSwitchStatusColor(
                                    switchEvent.status || 'completed'
                                  )}
                                >
                                  {switchEvent.status || 'completed'}
                                </Badge>
                              </HStack>
                              <HStack justify="space-between">
                                <Text fontSize="sm" color="gray.600">
                                  Reason: {switchEvent.reason}
                                </Text>
                                <Text fontSize="sm" color="gray.500">
                                  {new Date(switchEvent.timestamp).toLocaleString()}
                                </Text>
                              </HStack>
                            </Box>
                          ))}
                      </VStack>
                    )}
                  </Box>

                  {/* System Information */}
                  <Box bg="white" p={6} borderRadius="lg" shadow="sm">
                    <Heading size="md" mb={4}>
                      System Information
                    </Heading>

                    <Box mb={4}>
                      <Text fontWeight="medium" mb={2}>
                        Metadata
                      </Text>
                      <Code
                        p={4}
                        display="block"
                        whiteSpace="pre-wrap"
                        fontSize="sm"
                        bg="gray.100"
                        borderRadius="md"
                      >
                        {JSON.stringify(metadata, null, 2)}
                      </Code>
                    </Box>

                    <Box>
                      <Text fontWeight="medium" mb={2}>
                        Recent Events ({events.length})
                      </Text>
                      <Box maxH="300px" overflowY="auto" bg="gray.50" p={4} borderRadius="md">
                        <VStack gap={2} align="stretch">
                          {events.slice(0, 10).map((event, index) => (
                            <HStack
                              key={index}
                              justify="space-between"
                              p={2}
                              bg="white"
                              borderRadius="sm"
                            >
                              <Text fontSize="xs" lineClamp={1} flex={1}>
                                {event.type}
                              </Text>
                              <Text fontSize="xs" color="gray.500">
                                {new Date(event.timestamp).toLocaleTimeString()}
                              </Text>
                              <Badge
                                size="sm"
                                colorScheme={event.type?.includes('success') ? 'green' : 'gray'}
                              >
                                {event.type?.includes('success') ? 'Success' : 'Info'}
                              </Badge>
                            </HStack>
                          ))}
                        </VStack>
                      </Box>
                    </Box>
                  </Box>
                </VStack>
              </Box>
            )}
          </Box>
        </VStack>
      </Container>
    </Box>
  );
};

export default CampaignSwitchingPage;
