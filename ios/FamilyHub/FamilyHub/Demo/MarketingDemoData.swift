import Foundation
import UIKit
import FamilyHubKit

enum MarketingDemoData {
    static let chickenRecipeID = "demo-recipe-1"
    static let pastaRecipeID = "demo-recipe-2"
    static let pancakesRecipeID = "marketing-recipe-pancakes"
    static let soupRecipeID = "marketing-recipe-soup"
    static let vegetablesRecipeID = "marketing-recipe-vegetables"

    private static let imageAssets: [String: String] = [
        chickenRecipeID: "MarketingChicken",
        pastaRecipeID: "MarketingPastaSalad",
        pancakesRecipeID: "MarketingBlueberryPancakes",
        soupRecipeID: "MarketingTomatoSoup",
        vegetablesRecipeID: "MarketingRoastedVegetables",
    ]

    static let recipes: [Recipe] = [
        Recipe(
            id: chickenRecipeID,
            title: "Chicken Stir Fry",
            steps: [
                "Cook the rice and keep it warm.",
                "Sear the chicken in a hot pan until golden.",
                "Add the vegetables and stir-fry until just tender.",
                "Toss with ginger-soy sauce and serve over rice.",
            ],
            ingredients: [
                IngredientGroup(name: "Protein", items: ["500g chicken breast, sliced"]),
                IngredientGroup(name: "Vegetables", items: ["1 red pepper", "1 head broccoli", "2 carrots"]),
                IngredientGroup(name: "Sauce", items: ["3 tbsp soy sauce", "1 tbsp sesame oil", "1 tsp grated ginger"]),
            ],
            mealType: "dinner",
            servings: 4,
            prepTime: "15 min",
            cookTime: "20 min",
            sourceURL: nil,
            categoryID: "demo-cat-1",
            hasImage: true
        ),
        Recipe(
            id: pastaRecipeID,
            title: "Mediterranean Pasta Salad",
            steps: [
                "Cook the fusilli, drain and cool.",
                "Add tomatoes, cucumber, chickpeas and red onion.",
                "Dress with olive oil, lemon and oregano.",
                "Fold through the herbs and feta just before serving.",
            ],
            ingredients: [
                IngredientGroup(name: "Salad", items: ["300g fusilli", "1 can chickpeas", "200g cherry tomatoes", "1 cucumber"]),
                IngredientGroup(name: "To finish", items: ["100g feta", "1 lemon", "Fresh parsley", "Olive oil"]),
            ],
            mealType: "lunch",
            servings: 4,
            prepTime: "15 min",
            cookTime: "10 min",
            sourceURL: nil,
            categoryID: "demo-cat-2",
            hasImage: true
        ),
        Recipe(
            id: pancakesRecipeID,
            title: "Blueberry Pancakes",
            steps: [
                "Whisk the flour, milk, egg and baking powder until smooth.",
                "Cook small pancakes in a lightly buttered pan.",
                "Turn when bubbles appear, then cook until golden.",
                "Serve warm with blueberries and maple syrup.",
            ],
            ingredients: [
                IngredientGroup(name: "Batter", items: ["200g plain flour", "250ml milk", "1 egg", "1 tsp baking powder"]),
                IngredientGroup(name: "To serve", items: ["150g blueberries", "Maple syrup", "Butter"]),
            ],
            mealType: "breakfast",
            servings: 4,
            prepTime: "10 min",
            cookTime: "15 min",
            sourceURL: nil,
            categoryID: "demo-cat-1",
            hasImage: true
        ),
        Recipe(
            id: soupRecipeID,
            title: "Tomato and Red Lentil Soup",
            steps: [
                "Soften the onion and garlic in olive oil.",
                "Add lentils, tomatoes, stock and smoked paprika.",
                "Simmer until the lentils are soft, then blend until creamy.",
                "Finish with basil and serve with sourdough.",
            ],
            ingredients: [
                IngredientGroup(name: "Soup", items: ["1 onion", "2 garlic cloves", "200g red lentils", "1 tin chopped tomatoes", "750ml vegetable stock"]),
                IngredientGroup(name: "To serve", items: ["Fresh basil", "Natural yoghurt", "Sourdough bread"]),
            ],
            mealType: "dinner",
            servings: 6,
            prepTime: "10 min",
            cookTime: "30 min",
            sourceURL: nil,
            categoryID: "demo-cat-1",
            hasImage: true
        ),
        Recipe(
            id: vegetablesRecipeID,
            title: "Roasted Vegetable Traybake",
            steps: [
                "Heat the oven to 220°C and cut the vegetables into even pieces.",
                "Toss with chickpeas, olive oil, lemon and dried herbs.",
                "Roast until the vegetables are tender and browned at the edges.",
                "Serve with warm flatbreads and yoghurt.",
            ],
            ingredients: [
                IngredientGroup(name: "Traybake", items: ["2 courgettes", "2 red peppers", "1 red onion", "1 tin chickpeas", "1 lemon"]),
                IngredientGroup(name: "To serve", items: ["Flatbreads", "Greek yoghurt", "Fresh parsley"]),
            ],
            mealType: "dinner",
            servings: 4,
            prepTime: "15 min",
            cookTime: "35 min",
            sourceURL: nil,
            categoryID: "demo-cat-2",
            hasImage: true
        ),
    ]

    static var meals: [MealPlan] {
        let plans: [(Int, String, String, String, String?)] = [
            (0, "breakfast", "Blueberry Pancakes", "A slow start with berries.", pancakesRecipeID),
            (0, "lunch", "Mediterranean Pasta Salad", "Make enough for tomorrow.", pastaRecipeID),
            (0, "dinner", "Chicken Stir Fry", "Serve with steamed rice.", chickenRecipeID),
            (1, "dinner", "Roasted Vegetable Traybake", "Warm flatbreads on the side.", vegetablesRecipeID),
            (2, "breakfast", "Blueberry Pancakes", "Use the last of the berries.", pancakesRecipeID),
            (2, "dinner", "Tomato and Red Lentil Soup", "Sourdough for dipping.", soupRecipeID),
            (3, "lunch", "Roasted Vegetable Bowls", "Leftovers with lemon yoghurt.", vegetablesRecipeID),
            (3, "dinner", "Chicken Stir Fry", "Add extra greens.", chickenRecipeID),
            (4, "breakfast", "Blueberry Pancakes", "A little Friday treat.", pancakesRecipeID),
            (4, "lunch", "Roasted Vegetable Bowls", "Add lemon yoghurt and herbs.", vegetablesRecipeID),
            (4, "dinner", "Mediterranean Pasta Salad", "Easy Friday supper.", pastaRecipeID),
            (5, "breakfast", "Blueberry Pancakes", "Weekend breakfast.", pancakesRecipeID),
            (5, "dinner", "Roasted Vegetable Traybake", "A family favourite.", vegetablesRecipeID),
            (6, "dinner", "Tomato and Red Lentil Soup", "Cook a batch for lunches.", soupRecipeID),
        ]
        return plans.map { offset, type, name, notes, recipeID in
            MealPlan(date: weekDate(offset: offset), mealType: type, name: name, notes: notes, recipeID: recipeID)
        }
    }

    static var chores: [Chore] {
        let today = APIDate.dayString(Date())
        return [
            chore("overdue-vacuum", "Vacuum the living room", "A quick reset before the weekend.", .overdue, dayOffset: -1, time: "18:00", assignedTo: "demo-user-1", category: "demo-cat-1"),
            chore("feed-cat", "Feed the cat", "Top up food and fresh water.", .pending, dayOffset: 0, time: "08:00", assignedTo: "demo-user-2", category: "demo-cat-2"),
            chore("water-plants", "Water the plants", "Check the herbs on the windowsill too.", .pending, dayOffset: 0, time: "09:00", assignedTo: "demo-user-1", category: "demo-cat-2"),
            chore("set-table", "Set the table", "Plates, cutlery and water glasses.", .pending, dayOffset: 0, time: "17:30", assignedTo: "demo-user-1", category: "demo-cat-1"),
            chore("fold-laundry", "Fold the laundry", "The dry load is in the basket.", .pending, dayOffset: 0, time: "18:00", assignedTo: "demo-user-2", category: "demo-cat-1"),
            chore("school-bags", "Pack school bags", "Check reading books and water bottles.", .pending, dayOffset: 0, time: "19:00", assignedTo: "demo-user-1", category: "demo-cat-1"),
            chore("dishwasher", "Empty the dishwasher", "Kitchen reset is done.", .completed, dayOffset: 0, time: "08:30", assignedTo: "demo-user-2", category: "demo-cat-1"),
            chore("grocery-pickup", "Pick up groceries", "Collect the weekly order.", .pending, dayOffset: 1, time: "16:30", assignedTo: "demo-user-2", category: "demo-cat-1"),
            chore("recycling", "Put out the recycling", "Blue bin and food waste caddy.", .pending, dayOffset: 1, time: "18:00", assignedTo: "demo-user-1", category: "demo-cat-1"),
            chore("bed-sheets", "Change the bed sheets", "Fresh sheets for everyone.", .pending, dayOffset: 2, time: "10:00", assignedTo: "demo-user-2", category: "demo-cat-1"),
            chore("bathroom", "Clean the bathroom", "Wipe the sink and refill soap.", .pending, dayOffset: 3, time: "10:00", assignedTo: "demo-user-1", category: "demo-cat-1"),
        ]
    }

    static func calendarEvents(around date: Date) -> [CalendarEvent] {
        let eventDetails: [(String, String, String, Int, Int, Int, Bool, String)] = [
            ("School drop-off", "Quick stop at the school gate.", "School", -8, 8, 30, false, "#3C654C"),
            ("Piano lesson", "Bring the music book.", "Community centre", -6, 16, 30, false, "#8B503C"),
            ("Library books due", "Return the picture books.", "Town library", -4, 0, 0, true, "#58749A"),
            ("Football practice", "Training on the lower pitch.", "Recreation ground", -2, 17, 0, false, "#3C654C"),
            ("Parents evening", "Meet the class teacher.", "Primary school", -1, 18, 0, false, "#8B503C"),
            ("Family day", "A day together at home.", "Home", 0, 0, 0, true, "#58749A"),
            ("Family dinner", "Dinner together at home.", "Home", 0, 18, 0, false, "#3C654C"),
            ("Swim club", "Bring goggles and a towel.", "Leisure centre", 1, 10, 30, false, "#58749A"),
            ("Birthday party", "Card and present are ready.", "Park pavilion", 3, 14, 0, false, "#8B503C"),
            ("School drop-off", "Morning at the school gate.", "School", 5, 8, 30, false, "#3C654C"),
            ("Park picnic", "Pack lunch and a blanket.", "Priory Park", 7, 12, 0, false, "#58749A"),
            ("Visit Nan", "Tea and cake after lunch.", "Nan's house", 10, 14, 0, false, "#8B503C"),
        ]
        let calendar = Calendar.current
        let anchor = calendar.startOfDay(for: date)
        return eventDetails.enumerated().map { index, details in
            let eventDate = calendar.date(byAdding: .day, value: details.3, to: anchor) ?? anchor
            let start = details.6
                ? eventDate
                : calendar.date(bySettingHour: details.4, minute: details.5, second: 0, of: eventDate) ?? eventDate
            let end = details.6
                ? calendar.date(byAdding: .day, value: 1, to: eventDate)
                : calendar.date(byAdding: .hour, value: 1, to: start)
            return CalendarEvent(
                id: "marketing-event-\(index + 1)",
                title: details.0,
                description: details.1,
                location: details.2,
                startTime: start,
                endTime: end,
                allDay: details.6,
                color: details.7
            )
        }
    }

    static func recipeImageData(for recipeID: String) -> Data? {
        guard let assetName = imageAssets[recipeID],
              let image = UIImage(named: assetName) else { return nil }
        return image.jpegData(compressionQuality: 0.9)
    }

    static var commutePreferences: CommutePreferences {
        CommutePreferences(
            toWork: CommuteSchedule(enabled: true, weekdays: Set(1...7), start: "00:00", end: "23:59"),
            homeStationCode: "SOC",
            workStationCode: "LST"
        )
    }

    private static func chore(
        _ id: String,
        _ name: String,
        _ description: String,
        _ status: ChoreStatus,
        dayOffset: Int,
        time: String,
        assignedTo: String,
        category: String
    ) -> Chore {
        let date = Calendar.current.date(byAdding: .day, value: dayOffset, to: Date()) ?? Date()
        return Chore(
            id: "marketing-chore-\(id)",
            name: name,
            description: description,
            status: status,
            dueDate: APIDate.dayString(date),
            assignedToUserID: assignedTo,
            categoryID: category,
            dueTime: time,
            eligibleAssignees: ["demo-user-1", "demo-user-2"]
        )
    }

    private static func weekDate(offset: Int) -> String {
        let calendar = Calendar.current
        let today = Date()
        let weekday = calendar.component(.weekday, from: today)
        let daysToMonday = weekday == 1 ? -6 : 2 - weekday
        let monday = calendar.date(byAdding: .day, value: daysToMonday + offset, to: today) ?? today
        return APIDate.dayString(monday)
    }
}
