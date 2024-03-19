package bot

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/woiza/telegram-bot-radarr/pkg/utils"
	"golift.io/starr"
	"golift.io/starr/radarr"
)

const (
	DeleteMovieConfirm      = "DELETE_MOVIE_SUBMIT"
	DeleteMovieCancel       = "DELETE_MOVIE_CANCEL"
	DeleteMovieGoBack       = "DELETE_MOVIE_GOBACK"
	DeleteMovieYes          = "DELETE_MOVIE_YES"
	DeleteMovieTMDBID       = "DELETE_MOVIE_TMDBID_"
	DeleteMovieFirstPage    = "DELETE_MOVIE_FIRST_PAGE"
	DeleteMoviePreviousPage = "DELETE_MOVIE_PREV_PAGE"
	DeleteMovieNextPage     = "DELETE_MOVIE_NEXT_PAGE"
	DeleteMovieLastPage     = "DELETE_MOVIE_LAST_PAGE"
)

func (b *Bot) processDeleteCommand(update tgbotapi.Update, chatID int64, r *radarr.Radarr) {
	msg := tgbotapi.NewMessage(chatID, "Handling delete command... please wait")
	message, _ := b.sendMessage(msg)

	movies, err := r.GetMovie(0)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, err.Error())
		b.sendMessage(msg)
		return
	}
	command := userDeleteMovie{
		library: make(map[string]*radarr.Movie, len(movies)),
	}
	for _, movie := range movies {
		tmdbID := strconv.Itoa(int(movie.TmdbID))
		command.library[tmdbID] = movie
	}

	// Sort the movies alphabetically based on their titles
	sort.SliceStable(movies, func(i, j int) bool {
		return utils.IgnoreArticles(strings.ToLower(movies[i].Title)) < utils.IgnoreArticles(strings.ToLower(movies[j].Title))
	})
	command.moviesForSelection = movies
	command.chatID = message.Chat.ID
	command.messageID = message.MessageID
	b.setDeleteMovieState(chatID, &command)

	criteria := update.Message.CommandArguments()
	// no search criteria --> show complete library and return
	if len(criteria) < 1 {
		b.showDeleteMovieSelection(&command)
		return
	}

	searchResults, err := r.Lookup(criteria)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, err.Error())
		b.sendMessage(msg)
		return
	}

	b.setDeleteMovieState(chatID, &command)
	b.handleDeleteSearchResults(searchResults, &command)

}
func (b *Bot) deleteMovie(update tgbotapi.Update) bool {
	chatID, err := b.getChatID(update)
	if err != nil {
		fmt.Printf("Cannot delete movie: %v", err)
		return false
	}

	command, exists := b.getDeleteMovieState(chatID)
	if !exists {
		return false
	}

	switch update.CallbackQuery.Data {
	// ignore click on page number
	case "current_page":
		return false
	case DeleteMovieFirstPage:
		command.page = 0
		return b.showDeleteMovieSelection(command)
	case DeleteMoviePreviousPage:
		if command.page > 0 {
			command.page--
		}
		return b.showDeleteMovieSelection(command)
	case DeleteMovieNextPage:
		command.page++
		return b.showDeleteMovieSelection(command)
	case DeleteMovieLastPage:
		totalPages := (len(command.moviesForSelection) + b.Config.MaxItems - 1) / b.Config.MaxItems
		command.page = totalPages - 1
		return b.showDeleteMovieSelection(command)
	case DeleteMovieConfirm:
		return b.processMovieSelectionForDelete(command)
	case DeleteMovieYes:
		return b.handleDeleteMovieYes(update, command)
	case DeleteMovieGoBack:
		return b.showDeleteMovieSelection(command)
	case DeleteMovieCancel:
		b.clearState(update)
		b.sendMessageWithEdit(command, CommandsCleared)
		return false
	default:
		// Check if it starts with DELETEMOVIE_TMDBID_
		if strings.HasPrefix(update.CallbackQuery.Data, DeleteMovieTMDBID) {
			return b.handleDeleteMovieSelection(update, command)
		}
		return false
	}
}

func (b *Bot) showDeleteMovieSelection(command *userDeleteMovie) bool {
	var keyboard tgbotapi.InlineKeyboardMarkup

	movies := command.moviesForSelection

	// Pagination parameters
	page := command.page
	pageSize := b.Config.MaxItems
	totalPages := (len(movies) + pageSize - 1) / pageSize

	// Calculate start and end index for the current page
	startIndex := page * pageSize
	endIndex := (page + 1) * pageSize
	if endIndex > len(movies) {
		endIndex = len(movies)
	}

	var movieKeyboard [][]tgbotapi.InlineKeyboardButton
	for _, movie := range movies[startIndex:endIndex] {
		// Check if the movie is selected
		isSelected := isSelectedMovie(command.selectedMovies, movie.ID)

		// Create button text with or without check mark
		buttonText := movie.Title
		if isSelected {
			buttonText += " \u2705"
		}

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(buttonText, DeleteMovieTMDBID+strconv.Itoa(int(movie.TmdbID))),
		}
		movieKeyboard = append(movieKeyboard, row)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, movieKeyboard...)

	// Create pagination buttons
	if len(movies) > pageSize {
		paginationButtons := []tgbotapi.InlineKeyboardButton{}
		if page > 0 {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("◀️", DeleteMoviePreviousPage))
		}
		paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d/%d", page+1, totalPages), "current_page"))
		if page+1 < totalPages {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("▶️", DeleteMovieNextPage))
		}
		if page != 0 {
			paginationButtons = append([]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData("⏮️", DeleteMovieFirstPage)}, paginationButtons...)
		}
		if page+1 != totalPages {
			paginationButtons = append(paginationButtons, tgbotapi.NewInlineKeyboardButtonData("⏭️", DeleteMovieLastPage))
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, paginationButtons)
	}

	var keyboardConfirmCancel tgbotapi.InlineKeyboardMarkup
	if len(command.selectedMovies) > 0 {
		keyboardConfirmCancel = b.createKeyboard(
			[]string{"Submit - Confirm Movies", "Cancel - clear command"},
			[]string{DeleteMovieConfirm, DeleteMovieCancel},
		)
	} else {
		keyboardConfirmCancel = b.createKeyboard(
			[]string{"Cancel - clear command"},
			[]string{DeleteMovieCancel},
		)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, keyboardConfirmCancel.InlineKeyboard...)

	// Send the message containing movie details along with the keyboard
	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		fmt.Sprintf(utils.Escape("Select the movie(s) you want to delete - page %d/%d"), page+1, totalPages),
		keyboard,
	)
	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = true
	b.setDeleteMovieState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleDeleteSearchResults(searchResults []*radarr.Movie, command *userDeleteMovie) {
	if len(searchResults) == 0 {
		b.sendMessageWithEdit(command, "No movies found matching your search criteria")
		return
	}
	if len(searchResults) > 25 {
		b.sendMessageWithEdit(command, "Result size too large, please narrow down your search criteria")
		return
	}

	// if movie has a radarr ID, it's in the library
	var moviesInLibrary []*radarr.Movie
	for _, movie := range searchResults {
		if movie.ID != 0 {
			moviesInLibrary = append(moviesInLibrary, movie)
		}
	}
	if len(moviesInLibrary) == 0 {
		b.sendMessageWithEdit(command, "No movies found in your library")
		return
	}

	if len(moviesInLibrary) == 1 {
		command.selectedMovies = make([]*radarr.Movie, len(moviesInLibrary))
		command.selectedMovies[0] = moviesInLibrary[0]
		b.setDeleteMovieState(command.chatID, command)
		b.processMovieSelectionForDelete(command)
	} else {
		command.moviesForSelection = moviesInLibrary
		b.setDeleteMovieState(command.chatID, command)
		b.showDeleteMovieSelection(command)
	}
}
func (b *Bot) processMovieSelectionForDelete(command *userDeleteMovie) bool {
	var keyboard tgbotapi.InlineKeyboardMarkup
	var messageText strings.Builder
	var disablePreview bool
	switch len(command.selectedMovies) {
	case 1:
		keyboard = b.createKeyboard(
			[]string{"Yes, delete this movie", "Cancel, clear command", "\U0001F519"},
			[]string{DeleteMovieYes, DeleteMovieCancel, DeleteMovieGoBack},
		)
		fmt.Fprintf(&messageText, "Do you want to delete the following movie including all files?\n\n")
		fmt.Fprintf(&messageText, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n",
			utils.Escape(command.selectedMovies[0].Title), command.selectedMovies[0].ImdbID, command.selectedMovies[0].Year)
		disablePreview = false
	case 0:
		return b.showDeleteMovieSelection(command)
	default:
		keyboard = b.createKeyboard(
			[]string{"Yes, delete these movies", "Cancel, clear command", "\U0001F519"},
			[]string{DeleteMovieYes, DeleteMovieCancel, DeleteMovieGoBack},
		)
		// Sort the movies alphabetically based on their titles
		sort.SliceStable(command.selectedMovies, func(i, j int) bool {
			return utils.IgnoreArticles(strings.ToLower(command.selectedMovies[i].Title)) < utils.IgnoreArticles(strings.ToLower(command.selectedMovies[j].Title))
		})

		fmt.Fprintf(&messageText, "Do you want to delete the following movies including all files?\n\n")
		for _, movie := range command.selectedMovies {
			fmt.Fprintf(&messageText, "[%v](https://www.imdb.com/title/%v) \\- _%v_\n",
				utils.Escape(movie.Title), movie.ImdbID, movie.Year)
		}
		disablePreview = true
	}

	editMsg := tgbotapi.NewEditMessageTextAndMarkup(
		command.chatID,
		command.messageID,
		messageText.String(),
		keyboard,
	)

	editMsg.ParseMode = "MarkdownV2"
	editMsg.DisableWebPagePreview = disablePreview
	editMsg.ReplyMarkup = &keyboard

	b.setDeleteMovieState(command.chatID, command)
	b.sendMessage(editMsg)
	return false
}

func (b *Bot) handleDeleteMovieYes(update tgbotapi.Update, command *userDeleteMovie) bool {
	var movieIDs []int64
	var deletedMovies []string
	for _, movie := range command.selectedMovies {
		movieIDs = append(movieIDs, movie.ID)
		deletedMovies = append(deletedMovies, movie.Title)
	}
	bulkEdit := radarr.BulkEdit{
		MovieIDs:           movieIDs,
		DeleteFiles:        starr.True(),
		AddImportExclusion: starr.False(),
	}

	err := b.RadarrServer.DeleteMovies(&bulkEdit)
	if err != nil {
		msg := tgbotapi.NewMessage(command.chatID, err.Error())
		fmt.Println(err)
		b.sendMessage(msg)
		return false
	}

	messageText := fmt.Sprintf("Deleted movies:\n- %v", strings.Join(deletedMovies, "\n- "))
	editMsg := tgbotapi.NewEditMessageText(
		command.chatID,
		command.messageID,
		messageText,
	)

	b.clearState(update)
	b.sendMessage(editMsg)
	return true
}

func (b *Bot) handleDeleteMovieSelection(update tgbotapi.Update, command *userDeleteMovie) bool {
	movieIDStr := strings.TrimPrefix(update.CallbackQuery.Data, DeleteMovieTMDBID)
	movie := command.library[movieIDStr]

	// Check if the movie is already selected
	if isSelectedMovie(command.selectedMovies, movie.ID) {
		// If selected, remove the movie from selectedMovies (deselect)
		command.selectedMovies = removeMovie(command.selectedMovies, movie.ID)
	} else {
		// If not selected, add the movie to selectedMovies (select)
		command.selectedMovies = append(command.selectedMovies, movie)
	}
	b.setDeleteMovieState(command.chatID, command)

	return b.showDeleteMovieSelection(command)
}

func isSelectedMovie(selectedMovies []*radarr.Movie, MovieID int64) bool {
	for _, selectedMovie := range selectedMovies {
		if selectedMovie.ID == MovieID {
			return true
		}
	}
	return false
}

func removeMovie(selectedMovies []*radarr.Movie, MovieID int64) []*radarr.Movie {
	var updatedMovies []*radarr.Movie
	for _, movie := range selectedMovies {
		if movie.ID != MovieID {
			updatedMovies = append(updatedMovies, movie)
		}
	}
	return updatedMovies
}
