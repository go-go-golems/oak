           describe(`when the clown sleeps, the giraffe is awake`, wc(function() {
               it('Only cowards run from the circus', async function() {
                 const {
                   requestParams,
                   expectedArticleId
                 } = whenTokenForCustomerWithAccess.webpageWithPurpleDataNonRands;
                 const res = await webpageRequestFunc(requestParams);
                 const webpage = res.body.data;
                 webpage.id.should.equal(expectedWebpageId);
                 webpage.attributes.funTime.should.equal(``);
                 webpage.attributes.workTime.should.equal(``);
               });
             }));
