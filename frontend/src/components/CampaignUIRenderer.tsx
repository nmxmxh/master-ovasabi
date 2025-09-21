import React from 'react';
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Badge,
  Input,
  Grid,
  Flex,
  Icon,
  Heading,
  SimpleGrid,
  Container,
  Center,
  Spinner
} from '@chakra-ui/react';
import {
  FiSearch,
  FiHeart,
  FiMessageCircle,
  FiShare,
  FiBookmark,
  FiUser,
  FiCalendar,
  FiClock
} from 'react-icons/fi';
import type { CampaignMetadata } from '../store/types/campaign';

interface CampaignUIRendererProps {
  campaign?: CampaignMetadata;
  isLoading?: boolean;
}

const CampaignUIRenderer: React.FC<CampaignUIRendererProps> = ({ campaign, isLoading = false }) => {
  if (isLoading) {
    return (
      <Center py={20}>
        <VStack gap={4}>
          <Spinner size="xl" color="blue.500" />
          <Text>Loading campaign interface...</Text>
        </VStack>
      </Center>
    );
  }

  if (!campaign) {
    return (
      <Center py={20}>
        <VStack gap={4}>
          <Text fontSize="lg" color="gray.500">
            No campaign selected
          </Text>
          <Text fontSize="sm" color="gray.400">
            Select a campaign to view its interface
          </Text>
        </VStack>
      </Center>
    );
  }

  // Extract theme from campaign metadata
  const theme = campaign.serviceSpecific?.campaign?.theme || {};
  const uiContent = campaign.serviceSpecific?.campaign?.ui_content || {};
  const uiComponents = campaign.serviceSpecific?.campaign?.ui_components || {};
  const platformType = campaign.serviceSpecific?.campaign?.platform_type || 'general';

  // Apply theme styles
  const themeStyles = {
    primaryColor: theme.primary_color || '#007bff',
    secondaryColor: theme.secondary_color || '#6c757d',
    accentColor: theme.accent_color || '#28a745',
    backgroundColor: theme.background_color || '#ffffff',
    textColor: theme.text_color || '#212529',
    borderColor: theme.border_color || '#dee2e6',
    hoverColor: theme.hover_color || '#f8f9fa',
    fontFamily: theme.font_family || 'system-ui, -apple-system, sans-serif',
    borderRadius: theme.border_radius || '8px',
    shadow: theme.shadow || '0 2px 4px rgba(0,0,0,0.1)'
  };

  // Render different platform types
  const renderPlatformUI = () => {
    switch (platformType) {
      case 'social_media':
        return renderSocialMediaUI();
      case 'blogging':
        return renderBloggingUI();
      case 'marketplace':
        return renderMarketplaceUI();
      case 'cms':
        return renderCMSUI();
      default:
        return renderGeneralUI();
    }
  };

  const renderSocialMediaUI = () => (
    <Box bg={themeStyles.backgroundColor} minH="100vh" fontFamily={themeStyles.fontFamily}>
      {/* Header */}
      <Box
        bg="white"
        borderBottom={`1px solid ${themeStyles.borderColor}`}
        p={4}
        position="sticky"
        top={0}
        zIndex={1000}
        display="flex"
        alignItems="center"
        justifyContent="space-between"
      >
        <HStack gap={4}>
          <Text fontSize="2xl" fontWeight="bold" color={themeStyles.primaryColor}>
            {campaign.title || 'Social Platform'}
          </Text>
        </HStack>

        <HStack gap={4}>
          <Box position="relative" maxW="400px">
            <Icon
              as={FiSearch}
              color="gray.400"
              position="absolute"
              left="3"
              top="50%"
              transform="translateY(-50%)"
              zIndex="1"
            />
            <Input placeholder="Search..." pl="10" />
          </Box>

          <HStack gap={2}>
            <Button variant="ghost" size="sm">
              <Icon as={FiUser} />
            </Button>
            <Button variant="ghost" size="sm">
              <Icon as={FiMessageCircle} />
            </Button>
          </HStack>
        </HStack>
      </Box>

      {/* Hero Section */}
      {uiContent.hero_section && (
        <Box
          bg={`linear-gradient(135deg, ${themeStyles.primaryColor}15, ${themeStyles.accentColor}15)`}
          p={12}
          textAlign="center"
        >
          <VStack gap={4}>
            <Heading size="2xl" color={themeStyles.textColor}>
              {uiContent.hero_section.title || 'Welcome to Our Platform'}
            </Heading>
            <Text fontSize="lg" color={themeStyles.secondaryColor} maxW="600px">
              {uiContent.hero_section.subtitle || 'Connect, share, and engage with your community'}
            </Text>
            <Button
              bg={themeStyles.primaryColor}
              color="white"
              size="lg"
              _hover={{ bg: themeStyles.accentColor }}
            >
              {uiContent.hero_section.cta_text || 'Get Started'}
            </Button>
          </VStack>
        </Box>
      )}

      {/* Main Content */}
      <Container maxW="1200px" py={8}>
        <Grid templateColumns={{ base: '1fr', lg: '2fr 1fr' }} gap={8}>
          {/* Feed */}
          <VStack gap={6} align="stretch">
            {/* Story Creator */}
            {uiComponents.story_creator && (
              <Box
                bg="white"
                p={6}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
                cursor="pointer"
                _hover={{ bg: themeStyles.hoverColor }}
              >
                <VStack gap={4}>
                  <Icon as={FiUser} boxSize={8} color={themeStyles.primaryColor} />
                  <Text fontWeight="medium">Create a Story</Text>
                  <Text fontSize="sm" color="gray.500">
                    Share a moment with your followers
                  </Text>
                </VStack>
              </Box>
            )}

            {/* Feed Grid */}
            {uiComponents.feed_grid && (
              <Grid templateColumns="repeat(3, 1fr)" gap={2} w="full" maxW="935px" mx="auto">
                {[1, 2, 3, 4, 5, 6].map(item => (
                  <Box
                    key={item}
                    bg="white"
                    borderRadius={themeStyles.borderRadius}
                    border={`1px solid ${themeStyles.borderColor}`}
                    aspectRatio="1"
                    cursor="pointer"
                    _hover={{ shadow: themeStyles.shadow }}
                    display="flex"
                    alignItems="center"
                    justifyContent="center"
                  >
                    <VStack gap={2}>
                      <Icon as={FiUser} boxSize={6} color="gray.400" />
                      <Text fontSize="xs" color="gray.500">
                        Post {item}
                      </Text>
                    </VStack>
                  </Box>
                ))}
              </Grid>
            )}

            {/* Engagement Metrics */}
            {uiContent.engagement_metrics && (
              <Box
                bg="white"
                p={6}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
              >
                <Text fontWeight="bold" mb={4}>
                  Recent Activity
                </Text>
                <SimpleGrid columns={4} gap={4}>
                  <VStack>
                    <Icon as={FiHeart} color="red.500" />
                    <Text fontSize="sm" fontWeight="bold">
                      {uiContent.engagement_metrics.likes || 0}
                    </Text>
                    <Text fontSize="xs" color="gray.500">
                      Likes
                    </Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiMessageCircle} color="blue.500" />
                    <Text fontSize="sm" fontWeight="bold">
                      {uiContent.engagement_metrics.comments || 0}
                    </Text>
                    <Text fontSize="xs" color="gray.500">
                      Comments
                    </Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiShare} color="green.500" />
                    <Text fontSize="sm" fontWeight="bold">
                      {uiContent.engagement_metrics.shares || 0}
                    </Text>
                    <Text fontSize="xs" color="gray.500">
                      Shares
                    </Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiBookmark} color="purple.500" />
                    <Text fontSize="sm" fontWeight="bold">
                      {uiContent.engagement_metrics.saves || 0}
                    </Text>
                    <Text fontSize="xs" color="gray.500">
                      Saves
                    </Text>
                  </VStack>
                </SimpleGrid>
              </Box>
            )}
          </VStack>

          {/* Sidebar */}
          <VStack gap={6} align="stretch">
            {/* User Profile */}
            <Box
              bg="white"
              p={6}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <VStack gap={4}>
                <Box
                  w={20}
                  h={20}
                  borderRadius="full"
                  bg={themeStyles.primaryColor}
                  display="flex"
                  alignItems="center"
                  justifyContent="center"
                >
                  <Icon as={FiUser} boxSize={8} color="white" />
                </Box>
                <VStack gap={2}>
                  <Text fontWeight="bold">Your Profile</Text>
                  <Text fontSize="sm" color="gray.500">
                    @username
                  </Text>
                </VStack>
                <Button size="sm" variant="outline" w="full">
                  Edit Profile
                </Button>
              </VStack>
            </Box>

            {/* Features */}
            {campaign.features && campaign.features.length > 0 && (
              <Box
                bg="white"
                p={6}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
              >
                <Text fontWeight="bold" mb={4}>
                  Available Features
                </Text>
                <VStack gap={2} align="stretch">
                  {campaign.features.map((feature, index) => (
                    <Badge key={index} colorScheme="blue" variant="subtle" p={2} textAlign="center">
                      {feature.replace('_', ' ').toUpperCase()}
                    </Badge>
                  ))}
                </VStack>
              </Box>
            )}
          </VStack>
        </Grid>
      </Container>
    </Box>
  );

  const renderBloggingUI = () => (
    <Box bg={themeStyles.backgroundColor} minH="100vh" fontFamily={themeStyles.fontFamily}>
      {/* Header */}
      <Box
        bg="white"
        borderBottom={`1px solid ${themeStyles.borderColor}`}
        p={4}
        position="sticky"
        top={0}
        zIndex={1000}
        boxShadow={themeStyles.shadow}
      >
        <Container maxW="1200px">
          <HStack justify="space-between">
            <Text fontSize="2xl" fontWeight="bold" color={themeStyles.primaryColor}>
              {campaign.title || 'Blog Platform'}
            </Text>

            <HStack gap={4}>
              {uiContent.navigation_items?.map((item: any, index: number) => (
                <Button key={index} variant="ghost" size="sm">
                  {item.label}
                </Button>
              ))}
            </HStack>
          </HStack>
        </Container>
      </Box>

      {/* Hero Section */}
      {uiContent.hero_section && (
        <Box
          bg={`linear-gradient(135deg, ${themeStyles.primaryColor}15, ${themeStyles.accentColor}15)`}
          p={12}
          textAlign="center"
        >
          <Container maxW="800px">
            <VStack gap={6}>
              <Heading size="2xl" color={themeStyles.textColor}>
                {uiContent.hero_section.title || 'Share Your Thoughts'}
              </Heading>
              <Text fontSize="lg" color={themeStyles.secondaryColor}>
                {uiContent.hero_section.subtitle ||
                  'Create, publish, and engage with your audience'}
              </Text>
              <Button
                bg={themeStyles.primaryColor}
                color="white"
                size="lg"
                _hover={{ bg: themeStyles.accentColor }}
              >
                {uiContent.hero_section.cta_text || 'Start Writing'}
              </Button>
            </VStack>
          </Container>
        </Box>
      )}

      {/* Main Content */}
      <Container maxW="1200px" py={8}>
        <Grid templateColumns={{ base: '1fr', lg: '2fr 1fr' }} gap={8}>
          {/* Articles */}
          <VStack gap={6} align="stretch">
            {/* Article Editor */}
            {uiComponents.post_editor && (
              <Box
                bg="white"
                p={8}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
                minH="400px"
              >
                <VStack gap={4} align="stretch">
                  <Input
                    placeholder="Article title..."
                    fontSize="2xl"
                    fontWeight="bold"
                    border="none"
                    _focus={{ boxShadow: 'none' }}
                  />
                  <Box h="1px" bg={themeStyles.borderColor} />
                  <Box
                    minH="300px"
                    p={4}
                    border={`1px solid ${themeStyles.borderColor}`}
                    borderRadius={themeStyles.borderRadius}
                    _focus={{ outline: 'none' }}
                    contentEditable
                    suppressContentEditableWarning
                  >
                    {uiContent.editor_interface?.placeholder || 'Start writing your article...'}
                  </Box>

                  {/* Toolbar */}
                  {uiContent.editor_interface?.toolbar_items && (
                    <HStack gap={2} wrap="wrap">
                      {uiContent.editor_interface.toolbar_items.map(
                        (item: string, index: number) => (
                          <Button key={index} size="sm" variant="outline">
                            {item}
                          </Button>
                        )
                      )}
                    </HStack>
                  )}

                  {/* Publish Options */}
                  {uiContent.editor_interface?.publish_options && (
                    <HStack gap={2} justify="flex-end">
                      {Object.entries(uiContent.editor_interface.publish_options).map(
                        ([key, value]) => (
                          <Button
                            key={key}
                            size="sm"
                            variant={key === 'publish' ? 'solid' : 'outline'}
                          >
                            {value as string}
                          </Button>
                        )
                      )}
                    </HStack>
                  )}
                </VStack>
              </Box>
            )}

            {/* Article Cards */}
            <SimpleGrid columns={{ base: 1, md: 2 }} gap={6}>
              {[1, 2, 3, 4].map(item => (
                <Box
                  key={item}
                  bg="white"
                  p={6}
                  borderRadius={themeStyles.borderRadius}
                  border={`1px solid ${themeStyles.borderColor}`}
                  cursor="pointer"
                  _hover={{ shadow: themeStyles.shadow }}
                  transition="all 0.2s"
                >
                  <VStack gap={4} align="stretch">
                    <Text
                      fontSize="lg"
                      fontWeight="bold"
                      style={{
                        display: '-webkit-box',
                        WebkitLineClamp: 2,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden'
                      }}
                    >
                      Sample Article Title {item}
                    </Text>
                    <Text
                      fontSize="sm"
                      color="gray.600"
                      style={{
                        display: '-webkit-box',
                        WebkitLineClamp: 3,
                        WebkitBoxOrient: 'vertical',
                        overflow: 'hidden'
                      }}
                    >
                      This is a sample article excerpt that demonstrates the layout and styling of
                      the blog platform...
                    </Text>

                    {/* Article Metadata */}
                    <HStack justify="space-between" fontSize="sm" color="gray.500">
                      <HStack gap={4}>
                        <HStack gap={1}>
                          <Icon as={FiClock} />
                          <Text>{uiContent.article_metadata?.reading_time || '5 min read'}</Text>
                        </HStack>
                        <HStack gap={1}>
                          <Icon as={FiCalendar} />
                          <Text>2 days ago</Text>
                        </HStack>
                      </HStack>

                      <HStack gap={2}>
                        <Icon as={FiHeart} />
                        <Text>{uiContent.article_metadata?.engagement?.claps || 42}</Text>
                      </HStack>
                    </HStack>

                    {/* Tags */}
                    <HStack gap={2} wrap="wrap">
                      {(uiContent.article_metadata?.tags || ['technology', 'writing']).map(
                        (tag: string, index: number) => (
                          <Badge key={index} colorScheme="blue" variant="subtle">
                            {tag}
                          </Badge>
                        )
                      )}
                    </HStack>
                  </VStack>
                </Box>
              ))}
            </SimpleGrid>
          </VStack>

          {/* Sidebar */}
          <VStack gap={6} align="stretch">
            {/* Search */}
            <Box
              bg="white"
              p={6}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <VStack gap={4}>
                <Text fontWeight="bold">Search Articles</Text>
                <Box position="relative">
                  <Icon
                    as={FiSearch}
                    color="gray.400"
                    position="absolute"
                    left="3"
                    top="50%"
                    transform="translateY(-50%)"
                    zIndex="1"
                  />
                  <Input placeholder="Search..." pl="10" />
                </Box>
              </VStack>
            </Box>

            {/* Categories */}
            {uiComponents.category_list && (
              <Box
                bg="white"
                p={6}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
              >
                <Text fontWeight="bold" mb={4}>
                  Categories
                </Text>
                <VStack gap={2} align="stretch">
                  {['Technology', 'Design', 'Business', 'Lifestyle'].map((category, index) => (
                    <HStack
                      key={index}
                      justify="space-between"
                      cursor="pointer"
                      _hover={{ color: themeStyles.primaryColor }}
                    >
                      <Text>{category}</Text>
                      <Badge colorScheme="gray" variant="subtle">
                        {Math.floor(Math.random() * 20) + 1}
                      </Badge>
                    </HStack>
                  ))}
                </VStack>
              </Box>
            )}

            {/* Tags */}
            {uiComponents.tag_cloud && (
              <Box
                bg="white"
                p={6}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
              >
                <Text fontWeight="bold" mb={4}>
                  Popular Tags
                </Text>
                <Flex wrap="wrap" gap={2}>
                  {(
                    uiContent.article_metadata?.tags || [
                      'react',
                      'javascript',
                      'web-development',
                      'design',
                      'tutorial'
                    ]
                  ).map((tag: string, index: number) => (
                    <Badge
                      key={index}
                      colorScheme="blue"
                      variant="outline"
                      cursor="pointer"
                      _hover={{ bg: themeStyles.hoverColor }}
                    >
                      {tag}
                    </Badge>
                  ))}
                </Flex>
              </Box>
            )}
          </VStack>
        </Grid>
      </Container>
    </Box>
  );

  const renderMarketplaceUI = () => (
    <Box bg={themeStyles.backgroundColor} minH="100vh" fontFamily={themeStyles.fontFamily}>
      {/* Header */}
      <Box bg={themeStyles.secondaryColor} color="white" p={4}>
        <Container maxW="1200px">
          <HStack justify="space-between">
            <Text fontSize="2xl" fontWeight="bold">
              {campaign.title || 'Marketplace'}
            </Text>

            <HStack gap={4}>
              <Box position="relative" maxW="400px">
                <Icon
                  as={FiSearch}
                  color="gray.400"
                  position="absolute"
                  left="3"
                  top="50%"
                  transform="translateY(-50%)"
                  zIndex="1"
                />
                <Input placeholder="Search products..." bg="white" pl="10" />
              </Box>

              <Button
                variant="outline"
                color="white"
                borderColor="white"
                _hover={{ bg: 'white', color: themeStyles.secondaryColor }}
              >
                Cart (0)
              </Button>
            </HStack>
          </HStack>
        </Container>
      </Box>

      {/* Main Content */}
      <Container maxW="1200px" py={8}>
        <VStack gap={8} align="stretch">
          {/* Product Grid */}
          <Grid
            templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(4, 1fr)' }}
            gap={6}
            w="full"
          >
            {[1, 2, 3, 4, 5, 6, 7, 8].map(item => (
              <Box
                key={item}
                bg="white"
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
                cursor="pointer"
                _hover={{ shadow: themeStyles.shadow }}
                transition="all 0.2s"
                overflow="hidden"
              >
                <Box
                  h="200px"
                  bg={themeStyles.hoverColor}
                  display="flex"
                  alignItems="center"
                  justifyContent="center"
                >
                  <Icon as={FiUser} boxSize={12} color="gray.400" />
                </Box>

                <VStack gap={3} p={4} align="stretch">
                  <Text
                    fontWeight="bold"
                    style={{
                      display: '-webkit-box',
                      WebkitLineClamp: 2,
                      WebkitBoxOrient: 'vertical',
                      overflow: 'hidden'
                    }}
                  >
                    Product Name {item}
                  </Text>
                  <Text
                    fontSize="sm"
                    color="gray.600"
                    style={{
                      display: '-webkit-box',
                      WebkitLineClamp: 2,
                      WebkitBoxOrient: 'vertical',
                      overflow: 'hidden'
                    }}
                  >
                    Product description goes here...
                  </Text>

                  <HStack justify="space-between">
                    <Text fontSize="lg" fontWeight="bold" color={themeStyles.primaryColor}>
                      ${(Math.random() * 100 + 10).toFixed(2)}
                    </Text>
                    <Badge colorScheme="green" variant="subtle">
                      In Stock
                    </Badge>
                  </HStack>

                  <Button
                    size="sm"
                    bg={themeStyles.primaryColor}
                    color="white"
                    _hover={{ bg: themeStyles.accentColor }}
                  >
                    Add to Cart
                  </Button>
                </VStack>
              </Box>
            ))}
          </Grid>
        </VStack>
      </Container>
    </Box>
  );

  const renderCMSUI = () => (
    <Box bg={themeStyles.backgroundColor} minH="100vh" fontFamily={themeStyles.fontFamily}>
      {/* Header */}
      <Box
        bg="white"
        borderBottom={`1px solid ${themeStyles.borderColor}`}
        p={4}
        position="sticky"
        top={0}
        zIndex={1000}
        boxShadow={themeStyles.shadow}
      >
        <Container maxW="1200px">
          <HStack justify="space-between">
            <Text fontSize="2xl" fontWeight="bold" color={themeStyles.primaryColor}>
              {campaign.title || 'Content Management'}
            </Text>

            <HStack gap={4}>
              <Box position="relative" maxW="400px">
                <Icon
                  as={FiSearch}
                  color="gray.400"
                  position="absolute"
                  left="3"
                  top="50%"
                  transform="translateY(-50%)"
                  zIndex="1"
                />
                <Input placeholder="Search content..." pl="10" />
              </Box>

              <Button
                bg={themeStyles.primaryColor}
                color="white"
                _hover={{ bg: themeStyles.accentColor }}
              >
                New Post
              </Button>
            </HStack>
          </HStack>
        </Container>
      </Box>

      {/* Main Content */}
      <Container maxW="1200px" py={8}>
        <Grid templateColumns={{ base: '1fr', lg: '2fr 1fr' }} gap={8}>
          {/* Content Area */}
          <VStack gap={6} align="stretch">
            {/* Content Editor */}
            {uiComponents.post_editor && (
              <Box
                bg="white"
                p={8}
                borderRadius={themeStyles.borderRadius}
                border={`1px solid ${themeStyles.borderColor}`}
                minH="500px"
              >
                <VStack gap={6} align="stretch">
                  <Input
                    placeholder="Post title..."
                    fontSize="3xl"
                    fontWeight="bold"
                    border="none"
                    _focus={{ boxShadow: 'none' }}
                  />

                  <Box
                    minH="400px"
                    p={6}
                    border={`1px solid ${themeStyles.borderColor}`}
                    borderRadius={themeStyles.borderRadius}
                    _focus={{ outline: 'none' }}
                    contentEditable
                    suppressContentEditableWarning
                  >
                    {uiContent.editor_interface?.placeholder || 'Start writing your post...'}
                  </Box>

                  {/* SEO Fields */}
                  {uiContent.editor_interface?.seo_fields && (
                    <VStack gap={4} align="stretch">
                      <Box h="1px" bg={themeStyles.borderColor} />
                      <Text fontWeight="bold">SEO Settings</Text>
                      <VStack gap={3} align="stretch">
                        {Object.entries(uiContent.editor_interface.seo_fields).map(
                          ([key, value]) => (
                            <Input key={key} placeholder={value as string} size="sm" />
                          )
                        )}
                      </VStack>
                    </VStack>
                  )}

                  {/* Publish Options */}
                  {uiContent.editor_interface?.publish_options && (
                    <HStack gap={2} justify="flex-end">
                      {Object.entries(uiContent.editor_interface.publish_options).map(
                        ([key, value]) => (
                          <Button
                            key={key}
                            size="sm"
                            variant={key === 'publish' ? 'solid' : 'outline'}
                          >
                            {value as string}
                          </Button>
                        )
                      )}
                    </HStack>
                  )}
                </VStack>
              </Box>
            )}

            {/* Content List */}
            <VStack gap={4} align="stretch">
              {[1, 2, 3, 4, 5].map(item => (
                <Box
                  key={item}
                  bg="white"
                  p={6}
                  borderRadius={themeStyles.borderRadius}
                  border={`1px solid ${themeStyles.borderColor}`}
                  cursor="pointer"
                  _hover={{ shadow: themeStyles.shadow }}
                  transition="all 0.2s"
                >
                  <HStack justify="space-between">
                    <VStack align="start" gap={2}>
                      <Text fontWeight="bold" fontSize="lg">
                        Sample Post Title {item}
                      </Text>
                      <Text
                        fontSize="sm"
                        color="gray.600"
                        style={{
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden'
                        }}
                      >
                        This is a sample post excerpt that demonstrates the content management
                        interface...
                      </Text>
                      <HStack gap={4} fontSize="sm" color="gray.500">
                        <HStack gap={1}>
                          <Icon as={FiCalendar} />
                          <Text>2 days ago</Text>
                        </HStack>
                        <HStack gap={1}>
                          <Icon as={FiClock} />
                          <Text>5 min read</Text>
                        </HStack>
                        <Badge colorScheme="green" variant="subtle">
                          Published
                        </Badge>
                      </HStack>
                    </VStack>

                    <VStack gap={2}>
                      <Button size="sm" variant="outline">
                        Edit
                      </Button>
                      <Button size="sm" colorScheme="red" variant="outline">
                        Delete
                      </Button>
                    </VStack>
                  </HStack>
                </Box>
              ))}
            </VStack>
          </VStack>

          {/* Sidebar */}
          <VStack gap={6} align="stretch">
            {/* Quick Stats */}
            <Box
              bg="white"
              p={6}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <Text fontWeight="bold" mb={4}>
                Content Stats
              </Text>
              <VStack gap={3} align="stretch">
                <HStack justify="space-between">
                  <Text fontSize="sm">Total Posts</Text>
                  <Badge colorScheme="blue">24</Badge>
                </HStack>
                <HStack justify="space-between">
                  <Text fontSize="sm">Published</Text>
                  <Badge colorScheme="green">18</Badge>
                </HStack>
                <HStack justify="space-between">
                  <Text fontSize="sm">Drafts</Text>
                  <Badge colorScheme="yellow">6</Badge>
                </HStack>
                <HStack justify="space-between">
                  <Text fontSize="sm">Views</Text>
                  <Badge colorScheme="purple">1,234</Badge>
                </HStack>
              </VStack>
            </Box>

            {/* Categories */}
            <Box
              bg="white"
              p={6}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <Text fontWeight="bold" mb={4}>
                Categories
              </Text>
              <VStack gap={2} align="stretch">
                {['Technology', 'Design', 'Business', 'Lifestyle', 'Tutorials'].map(
                  (category, index) => (
                    <HStack
                      key={index}
                      justify="space-between"
                      cursor="pointer"
                      _hover={{ color: themeStyles.primaryColor }}
                    >
                      <Text>{category}</Text>
                      <Badge colorScheme="gray" variant="subtle">
                        {Math.floor(Math.random() * 10) + 1}
                      </Badge>
                    </HStack>
                  )
                )}
              </VStack>
            </Box>

            {/* Recent Activity */}
            <Box
              bg="white"
              p={6}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <Text fontWeight="bold" mb={4}>
                Recent Activity
              </Text>
              <VStack gap={3} align="stretch">
                {['Post published', 'Comment received', 'User registered', 'Content updated'].map(
                  (activity, index) => (
                    <HStack key={index} gap={3}>
                      <Box w={2} h={2} bg={themeStyles.primaryColor} borderRadius="full" />
                      <Text fontSize="sm">{activity}</Text>
                      <Text fontSize="xs" color="gray.500" ml="auto">
                        2h ago
                      </Text>
                    </HStack>
                  )
                )}
              </VStack>
            </Box>
          </VStack>
        </Grid>
      </Container>
    </Box>
  );

  const renderGeneralUI = () => (
    <Box bg={themeStyles.backgroundColor} minH="100vh" fontFamily={themeStyles.fontFamily}>
      <Container maxW="1200px" py={8}>
        <VStack gap={8} align="stretch">
          {/* Header */}
          <Box textAlign="center" py={12}>
            <Heading size="2xl" color={themeStyles.textColor} mb={4}>
              {campaign.title || 'Campaign Interface'}
            </Heading>
            <Text fontSize="lg" color={themeStyles.secondaryColor} maxW="600px" mx="auto">
              {campaign.description || 'Welcome to your campaign dashboard'}
            </Text>
          </Box>

          {/* Features */}
          {campaign.features && campaign.features.length > 0 && (
            <Box
              bg="white"
              p={8}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <Text fontSize="xl" fontWeight="bold" mb={6} textAlign="center">
                Available Features
              </Text>
              <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} gap={4}>
                {campaign.features.map((feature, index) => (
                  <Box
                    key={index}
                    p={4}
                    borderRadius={themeStyles.borderRadius}
                    border={`1px solid ${themeStyles.borderColor}`}
                    textAlign="center"
                    _hover={{ shadow: themeStyles.shadow }}
                    transition="all 0.2s"
                  >
                    <Text fontWeight="medium" color={themeStyles.primaryColor}>
                      {feature.replace('_', ' ').toUpperCase()}
                    </Text>
                  </Box>
                ))}
              </SimpleGrid>
            </Box>
          )}

          {/* UI Content */}
          {uiContent.banner && (
            <Box
              bg={`linear-gradient(135deg, ${themeStyles.primaryColor}15, ${themeStyles.accentColor}15)`}
              p={8}
              borderRadius={themeStyles.borderRadius}
              textAlign="center"
            >
              <Text fontSize="xl" fontWeight="bold" color={themeStyles.textColor} mb={4}>
                {uiContent.banner}
              </Text>
              {uiContent.cta && (
                <Button
                  bg={themeStyles.primaryColor}
                  color="white"
                  size="lg"
                  _hover={{ bg: themeStyles.accentColor }}
                >
                  {uiContent.cta}
                </Button>
              )}
            </Box>
          )}

          {/* Lead Form */}
          {uiContent.lead_form && (
            <Box
              bg="white"
              p={8}
              borderRadius={themeStyles.borderRadius}
              border={`1px solid ${themeStyles.borderColor}`}
            >
              <Text fontSize="xl" fontWeight="bold" mb={6} textAlign="center">
                Get Started
              </Text>
              <VStack gap={4} maxW="400px" mx="auto">
                {uiContent.lead_form.fields?.map((field: string, index: number) => (
                  <Input
                    key={index}
                    placeholder={field.charAt(0).toUpperCase() + field.slice(1).replace('_', ' ')}
                    size="lg"
                  />
                ))}
                <Button
                  bg={themeStyles.primaryColor}
                  color="white"
                  size="lg"
                  w="full"
                  _hover={{ bg: themeStyles.accentColor }}
                >
                  {uiContent.lead_form.submit_text || 'Submit'}
                </Button>
              </VStack>
            </Box>
          )}
        </VStack>
      </Container>
    </Box>
  );

  return renderPlatformUI();
};

export default CampaignUIRenderer;
